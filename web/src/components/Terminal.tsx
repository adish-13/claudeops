import { useEffect, useRef } from "react";
import { Terminal as Xterm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";

export function Terminal({ workspaceId }: { workspaceId: number }) {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current) return;
    const term = new Xterm({
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      fontSize: 13,
      theme: { background: "#000000", foreground: "#e6edf3", cursor: "#58a6ff" },
      cursorBlink: true,
      scrollback: 5000,
      allowProposedApi: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(hostRef.current);
    fit.fit();

    const proto = location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${location.host}/ws/terminal/${workspaceId}`);
    ws.binaryType = "arraybuffer";

    const dec = new TextDecoder();
    ws.onmessage = (ev) => {
      const data = ev.data instanceof ArrayBuffer ? dec.decode(new Uint8Array(ev.data)) : ev.data;
      term.write(data);
    };
    ws.onopen = () => {
      ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
      term.focus();
    };

    const inputDispose = term.onData((d) => {
      if (ws.readyState === 1) ws.send(JSON.stringify({ type: "input", data: d }));
    });
    const onResize = () => {
      try {
        fit.fit();
      } catch {}
      if (ws.readyState === 1) ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
    };
    window.addEventListener("resize", onResize);

    return () => {
      window.removeEventListener("resize", onResize);
      inputDispose.dispose();
      ws.close();
      term.dispose();
    };
  }, [workspaceId]);

  return <div ref={hostRef} className="h-full w-full" />;
}
