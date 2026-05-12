import { useEffect, useRef } from "react";
import { Terminal as Xterm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";

export function Terminal({ workspaceId }: { workspaceId: number }) {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current) return;
    const host = hostRef.current;
    const term = new Xterm({
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
      fontSize: 13,
      // Pin lineHeight/letterSpacing so xterm's cell math matches what the
      // browser actually renders — relying on defaults here can leave a tiny
      // sub-pixel mismatch that compounds into visible row overlap.
      lineHeight: 1.0,
      letterSpacing: 0,
      theme: { background: "#000000", foreground: "#e6edf3", cursor: "#58a6ff" },
      cursorBlink: true,
      scrollback: 5000,
      allowProposedApi: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);

    const proto = location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${location.host}/ws/terminal/${workspaceId}`);
    ws.binaryType = "arraybuffer";

    let lastRows = 0;
    let lastCols = 0;
    const refit = () => {
      if (host.clientWidth === 0 || host.clientHeight === 0) return;
      try {
        fit.fit();
      } catch {
        return;
      }
      if ((term.rows !== lastRows || term.cols !== lastCols) && ws.readyState === 1) {
        lastRows = term.rows;
        lastCols = term.cols;
        ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
      }
    };

    // Defer the initial fit until layout has settled so the host has a real size.
    const raf = requestAnimationFrame(refit);

    // xterm measures the cell width with whatever font is loaded at construction
    // time. If `ui-monospace` resolves later than the system fallback, the cached
    // cell width is wrong and glyphs overlap. Refit once fonts are ready, and
    // also force a re-render so cached measurements are thrown out.
    let fontsCancelled = false;
    if (document.fonts && document.fonts.ready) {
      document.fonts.ready.then(() => {
        if (fontsCancelled) return;
        try {
          term.clearTextureAtlas?.();
        } catch {}
        refit();
      });
    }

    const dec = new TextDecoder();
    ws.onmessage = (ev) => {
      const data = ev.data instanceof ArrayBuffer ? dec.decode(new Uint8Array(ev.data)) : ev.data;
      term.write(data);
    };
    ws.onopen = () => {
      refit();
      // Always send the current size on connect, even if it matches lastRows/lastCols
      // (which start at 0) — the server-side pty needs to know the viewport.
      ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
      lastRows = term.rows;
      lastCols = term.cols;
      term.focus();
    };

    const inputDispose = term.onData((d) => {
      if (ws.readyState === 1) ws.send(JSON.stringify({ type: "input", data: d }));
    });

    const ro = new ResizeObserver(refit);
    ro.observe(host);

    return () => {
      fontsCancelled = true;
      cancelAnimationFrame(raf);
      ro.disconnect();
      inputDispose.dispose();
      ws.close();
      term.dispose();
    };
  }, [workspaceId]);

  return <div ref={hostRef} className="h-full w-full" />;
}
