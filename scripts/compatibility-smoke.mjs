// Opt-in integration test against an explicitly supplied t3@0.0.40 npm entrypoint.
// All server state and credentials belong to this test. Never targets a running install.
import { spawn } from "node:child_process";
import { randomBytes } from "node:crypto";
import { mkdtemp, mkdir, readFile, writeFile, rm } from "node:fs/promises";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { setTimeout as delay } from "node:timers/promises";

const entrypoint = process.argv[2] ? resolve(process.argv[2]) : null;
if (!entrypoint) {
    console.error("Usage: node scripts/compatibility-smoke.mjs /path/to/t3/dist/bin.mjs [collector-binary]");
    process.exit(2);
}
const collector = process.argv[3] ? resolve(process.argv[3]) : null;
let phase = "verify supplied release";
let directory;
let server;
let socket;
const passed = (label) => console.log(`PASS ${label}`);
const check = (condition) => { if (!condition) throw new Error("check-failed"); };

async function stopServer() {
    socket?.close();
    socket = undefined;
    if (!server?.pid || server.exitCode !== null || server.signalCode !== null) return;
    const exited = new Promise(resolveExit => server.once("exit", resolveExit));
    server.kill("SIGTERM");
    await Promise.race([exited, delay(3000)]);
    if (server.exitCode === null && server.signalCode === null) {
        server.kill("SIGKILL");
        await exited;
    }
}

try {
    const manifest = JSON.parse(await readFile(join(dirname(entrypoint), "..", "package.json"), "utf8"));
    check(manifest.name === "t3" && manifest.version === "0.0.40");
    directory = await mkdtemp(join(tmpdir(), "dankt3code-compatibility-"));
    await mkdir(join(directory, "userdata"));
    await mkdir(join(directory, "bin"));
    await writeFile(join(directory, "userdata", "settings.json"), JSON.stringify({
        providers: Object.fromEntries(["codex", "claudeAgent", "cursor", "grok", "opencode", "antigravity"].map(driver => [driver, { enabled: false }])),
        providerInstances: {},
    }), { mode: 0o600 });

    const listener = createServer();
    await new Promise(resolveListen => listener.listen(0, "127.0.0.1", resolveListen));
    const port = listener.address().port;
    await new Promise(resolveClose => listener.close(resolveClose));
    const origin = `http://127.0.0.1:${port}`;
    const bootstrap = randomBytes(32).toString("hex");
    const clientLabel = "DankT3Code synthetic smoke client";
    const setupScopes = "orchestration:read orchestration:operate terminal:operate review:write relay:read access:read access:write";
    // Both subprocesses get test-only state and no inherited credentials or provider binaries.
    const isolatedEnvironment = {
        HOME: directory, PATH: join(directory, "bin"),
        XDG_CONFIG_HOME: join(directory, "config"), XDG_CACHE_HOME: join(directory, "cache"), XDG_DATA_HOME: join(directory, "data"),
        NODE_NO_WARNINGS: "1", T3CODE_TELEMETRY_ENABLED: "false",
    };

    async function request(path, { token, body, method = "GET", timeout = 5000 } = {}) {
        const headers = {};
        if (token) headers.Authorization = `Bearer ${token}`;
        if (body && !(body instanceof URLSearchParams)) headers["Content-Type"] = "application/json";
        const response = await fetch(origin + path, {
            method, headers, redirect: "error", signal: AbortSignal.timeout(timeout),
            body: body instanceof URLSearchParams ? body : body ? JSON.stringify(body) : undefined,
        });
        return { status: response.status, value: await response.json() };
    }

    async function startServer() {
        server = spawn(process.execPath, [entrypoint, "--bootstrap-fd", "0", "--log-level", "none"], {
            cwd: directory,
            env: isolatedEnvironment,
            stdio: ["pipe", "ignore", "ignore"],
        });
        let spawnFailed = false;
        server.on("error", () => { spawnFailed = true; });
        server.stdin.on("error", () => {});
        server.stdin.end(JSON.stringify({
            mode: "desktop", noBrowser: true, port, host: "127.0.0.1", t3Home: directory,
            desktopBootstrapToken: bootstrap, tailscaleServeEnabled: false, tailscaleServePort: 443,
        }) + "\n");
        const deadline = performance.now() + 10000;
        while (performance.now() < deadline) {
            check(!spawnFailed && server.exitCode === null && server.signalCode === null);
            try {
                const result = await request("/.well-known/t3/environment", { timeout: 500 });
                if (result.status === 200) return result.value;
            } catch {}
            await delay(100);
        }
        throw new Error("startup-timeout");
    }

    async function exchange(credential, scope, label) {
        return request("/oauth/token", { method: "POST", body: new URLSearchParams({
            grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
            subject_token: credential,
            subject_token_type: "urn:t3:params:oauth:token-type:environment-bootstrap",
            requested_token_type: "urn:ietf:params:oauth:token-type:access_token",
            scope, client_label: label,
        }) });
    }

    async function collect(token) {
        const tokenFile = join(directory, "collector-token");
        await writeFile(tokenFile, token + "\n", { mode: 0o600 });
        const result = await new Promise((resolveResult, rejectResult) => {
            const process = spawn(collector, ["snapshot", "--endpoint", origin, "--token-file", tokenFile], {
                env: isolatedEnvironment, cwd: directory, stdio: ["ignore", "pipe", "ignore"], timeout: 15000,
            });
            let output = "";
            process.stdout.on("data", data => { if (output.length < 65536) output += data.toString(); });
            process.on("error", rejectResult);
            process.on("close", code => resolveResult({ code, output }));
        });
        check(!result.output.includes(token));
        return { code: result.code, value: JSON.parse(result.output) };
    }

    phase = "isolated released-server startup";
    const descriptor = await startServer();
    check(descriptor.serverVersion === "0.0.40" && typeof descriptor.environmentId === "string");
    passed(phase);

    phase = "one-time pairing narrowed to observation scope";
    let admin = await exchange(bootstrap, setupScopes, "Synthetic setup");
    check(admin.status === 200);
    // A normal pairing token can be narrowed at exchange, as the production client will do.
    const pairing = await request("/api/auth/pairing-token", { method: "POST", token: admin.value.access_token, body: { label: clientLabel } });
    check(pairing.status === 200);
    const paired = await exchange(pairing.value.credential, "orchestration:read", clientLabel);
    check(paired.status === 200 && paired.value.scope === "orchestration:read" && paired.value.token_type === "Bearer");
    const token = paired.value.access_token;
    const replay = await exchange(pairing.value.credential, "orchestration:read", clientLabel);
    check(replay.status === 401);
    passed(phase);

    phase = "HTTP shell and unauthorized-client rejection";
    let shell = await request("/api/orchestration/shell", { token });
    check(shell.status === 200 && shell.value.projects.length === 0 && shell.value.threads.length === 0 && Number.isInteger(shell.value.snapshotSequence));
    check((await request("/api/orchestration/shell")).status === 401);
    passed(phase);

    phase = "read-only scope denies access administration";
    const denied = await request("/api/auth/pairing-token", { token, method: "POST", body: {} });
    check(denied.status === 403 && denied.value.code === "insufficient_scope" && denied.value.requiredScope === "access:write");
    check((await request("/api/auth/clients", { token })).status === 403);
    const deniedCommand = await request("/api/orchestration/dispatch", { token, method: "POST", body: {
        type: "thread.archive", commandId: "synthetic-denied-command", threadId: "synthetic-absent-thread", createdAt: "2026-09-13T12:00:00.000Z",
    } });
    check(deniedCommand.status === 403 && deniedCommand.value.requiredScope === "orchestration:operate");
    passed(phase);

    phase = "ticket-authenticated WebSocket snapshot";
    const ticket = await request("/api/auth/websocket-ticket", { token, method: "POST" });
    check(ticket.status === 200);
    socket = new WebSocket(`ws://127.0.0.1:${port}/ws?wsTicket=${encodeURIComponent(ticket.value.ticket)}`);
    let snapshotSeen = false;
    let synchronized = false;
    socket.addEventListener("message", event => {
        let decoded;
        try { decoded = JSON.parse(event.data); } catch { return; }
        for (const message of Array.isArray(decoded) ? decoded : [decoded]) {
            if (message._tag === "Chunk" && String(message.requestId) === "1") {
                for (const value of message.values ?? []) {
                    if (value.kind === "snapshot" && value.snapshot?.threads?.length === 0
                        && value.snapshot?.projects?.length === 0
                        && value.snapshot?.snapshotSequence === shell.value.snapshotSequence) snapshotSeen = true;
                    if (value.kind === "synchronized") synchronized = true;
                }
                socket.send(JSON.stringify({ _tag: "Ack", requestId: message.requestId }));
            }
            if (message._tag === "Ping") socket.send(JSON.stringify({ _tag: "Pong" }));
        }
    });
    await Promise.race([
        new Promise((resolveOpen, rejectOpen) => {
            socket.addEventListener("open", resolveOpen, { once: true });
            socket.addEventListener("error", () => rejectOpen(new Error("socket-failed")), { once: true });
        }),
        delay(5000).then(() => { throw new Error("socket-timeout"); }),
    ]);
    socket.send(JSON.stringify({ _tag: "Request", id: "1", tag: "orchestration.subscribeShell", payload: { requestCompletionMarker: true }, headers: [] }));
    for (let attempt = 0; attempt < 50 && !(snapshotSeen && synchronized); attempt++) await delay(100);
    check(snapshotSeen && synchronized);
    passed(phase);

    if (collector) {
        phase = "Go collector against released HTTP server";
        const result = await collect(token);
        check(result.code === 0);
        const output = result.value;
        check(output.schemaVersion === 1 && output.state === "connected" && output.environment?.id === descriptor.environmentId
            && output.error === null && output.snapshot?.threads?.length === 0 && output.snapshot?.projects?.length === 0);
        passed(phase);
    }

    phase = "restart preserves environment identity and paired bearer";
    await stopServer();
    const restarted = await startServer();
    check(restarted.environmentId === descriptor.environmentId);
    shell = await request("/api/orchestration/shell", { token });
    check(shell.status === 200 && shell.value.threads.length === 0);
    if (collector) {
        const result = await collect(token);
        check(result.code === 0 && result.value.state === "connected" && result.value.environment?.id === descriptor.environmentId);
    }
    passed(phase);

    phase = "revocation rejects subsequent HTTP collection";
    admin = await exchange(bootstrap, setupScopes, "Synthetic setup");
    check(admin.status === 200);
    const clients = await request("/api/auth/clients", { token: admin.value.access_token });
    check(clients.status === 200);
    const client = clients.value.find(value => value.client?.label === clientLabel);
    check(client && !client.current);
    const revoked = await request("/api/auth/clients/revoke", { token: admin.value.access_token, method: "POST", body: { sessionId: client.sessionId } });
    check(revoked.status === 200 && revoked.value.revoked === true);
    check((await request("/api/orchestration/shell", { token })).status === 401);
    if (collector) {
        const result = await collect(token);
        check(result.code === 1 && result.value.state === "error" && result.value.snapshot === null
            && result.value.error?.code === "authentication_failed");
    }
    passed(phase);
} catch {
    // A raw error or response can contain credentials. Only fixed phase labels leave the test.
    console.error(`FAIL ${phase}`);
    process.exitCode = 1;
} finally {
    await stopServer();
    if (directory) await rm(directory, { recursive: true, force: true });
}
