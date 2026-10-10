const $ = (sel) => document.querySelector(sel);
const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
const fmtInt = (n) => n.toLocaleString("en-US");

function fmtKB(kb) {
  if (kb >= 1024 * 1024) return (kb / 1024 / 1024).toFixed(1) + " GB";
  if (kb >= 1024) return (kb / 1024).toFixed(kb >= 10 * 1024 ? 0 : 1) + " MB";
  return kb + " KB";
}

function fmtUptime(s) {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
  return d ? `${d}d ${h}h` : h ? `${h}h ${m}m` : `${m}m`;
}

const STATES = {
  R: "running", S: "sleeping", D: "disk sleep", T: "stopped", t: "traced",
  Z: "zombie", X: "dead", I: "idle", P: "parked",
};

/* ---------------------------------------------------------------- tooltip */
const tip = $("#tip");
document.addEventListener("pointermove", (e) => {
  const el = e.target.closest("[data-tip]");
  if (!el || !el.dataset.tip) { tip.hidden = true; return; }
  tip.innerHTML = el.dataset.tip;
  tip.hidden = false;
  const pad = 14, w = tip.offsetWidth, h = tip.offsetHeight;
  let x = e.clientX + pad, y = e.clientY + pad;
  if (x + w > innerWidth - 8) x = e.clientX - w - pad;
  if (y + h > innerHeight - 8) y = e.clientY - h - pad;
  tip.style.left = x + "px";
  tip.style.top = y + "px";
});
addEventListener("scroll", () => (tip.hidden = true), { passive: true });

/* ---------------------------------------------------------- data loading */
// Served by `kmon serve` → live API. On GitHub Pages → the snapshot taken in CI.
let snapshot = null;
let live = false;
const collapsed = new Set();

async function load() {
  try {
    const r = await fetch("api/snapshot", { cache: "no-store" });
    if (r.ok && (r.headers.get("content-type") || "").includes("json")) {
      live = true;
      return r.json();
    }
  } catch { /* not running behind kmon serve */ }
  const r = await fetch("snapshot.json", { cache: "no-cache" });
  if (!r.ok) throw new Error("snapshot.json not found");
  return r.json();
}

async function refresh() {
  try {
    snapshot = await load();
    render();
  } catch (err) {
    $("#sourceText").textContent = "No data: load the module and run `kmon serve`, or add web/snapshot.json.";
  }
  if (live) setTimeout(refresh, 2000);
}

/* --------------------------------------------------------------- render */
function render() {
  const s = snapshot, h = s.host;
  const when = new Date(s.taken_at).toLocaleString("en-US", { dateStyle: "medium", timeStyle: "short" });
  $("#source").classList.toggle("live", live);
  $("#sourceText").textContent = live
    ? `Live · kernel ${h.kernel} · ${h.cpus} CPUs · up ${fmtUptime(h.uptime_seconds)}`
    : `Snapshot of a GitHub Actions VM · kernel ${h.kernel} · ${h.cpus} CPUs · ${when}`;

  // memory
  const m = s.memory, used = Math.max(m.total_kb - m.available_kb, 0), pct = (used / m.total_kb) * 100;
  $("#memPct").textContent = pct.toFixed(0) + "%";
  $("#memFill").style.width = pct + "%";
  $("#memSub").textContent = `${fmtKB(used)} of ${fmtKB(m.total_kb)} · ${fmtKB(m.available_kb)} available`;
  $("#memMeter").dataset.tip = `<b>Memory</b><br>in use ${fmtKB(used)}<br>available ${fmtKB(m.available_kb)}<br>free ${fmtKB(m.free_kb)} · buffers ${fmtKB(m.buffers_kb)} · shared ${fmtKB(m.shared_kb)}`;

  // processes + states
  const st = s.states;
  const kthreads = s.processes.filter(isKernelThread).length;
  $("#procTotal").textContent = fmtInt(st.total);
  $("#procSub").textContent = `${fmtInt(st.total - kthreads)} user · ${fmtInt(kthreads)} kernel threads`;
  const items = [["Running", st.running], ["Sleeping", st.sleeping], ["Idle", st.idle],
                 ["Disk sleep", st.disk_sleep], ["Stopped", st.stopped], ["Zombie", st.zombie]];
  $("#states").innerHTML = items.map(([k, v]) =>
    `<div><dt>${k}</dt><dd class="${k === "Zombie" && v > 0 ? "alert" : ""}">${fmtInt(v)}</dd></div>`).join("");

  renderTopMemory();
  renderTree();
}

// Kernel threads are children of kthreadd (PID 2) and have no user memory.
function isKernelThread(p) {
  return p.pid === 2 || (p.ppid === 2 && p.rss_kb === 0);
}

function renderTopMemory() {
  const top = [...snapshot.processes].filter((p) => p.rss_kb > 0).sort((a, b) => b.rss_kb - a.rss_kb).slice(0, 8);
  const max = top[0]?.rss_kb || 1;
  $("#topMem").innerHTML = top.map((p) => {
    const t = `<b>${esc(p.name)}</b> · PID ${p.pid}<br>${fmtKB(p.rss_kb)} resident · ${p.threads} thread${p.threads === 1 ? "" : "s"}`;
    return `<span class="lbl">${esc(p.name)} <small>${p.pid}</small></span>
      <div class="track" data-tip="${esc(t)}"><div class="bar" style="width:${(p.rss_kb / max) * 100}%"></div></div>
      <span class="val">${fmtKB(p.rss_kb)}</span>`;
  }).join("");
}

function renderTree() {
  const q = $("#search").value.trim().toLowerCase();
  const showKernel = $("#showKernel").checked;
  const procs = snapshot.processes.filter((p) => showKernel || !isKernelThread(p));
  const byPid = new Map(procs.map((p) => [p.pid, p]));
  const children = new Map();
  for (const p of procs) {
    const parent = byPid.has(p.ppid) ? p.ppid : 0;
    if (!children.has(parent)) children.set(parent, []);
    children.get(parent).push(p);
  }

  // With a filter, show matches together with their ancestors so the tree stays readable.
  let visible = null;
  if (q) {
    visible = new Set();
    for (const p of procs) {
      if (p.name.toLowerCase().includes(q) || String(p.pid) === q) {
        for (let cur = p; cur; cur = byPid.get(cur.ppid)) visible.add(cur.pid);
      }
    }
  }

  const rows = [];
  const walk = (parentPid, depth) => {
    for (const p of children.get(parentPid) || []) {
      if (visible && !visible.has(p.pid)) continue;
      const kids = children.get(p.pid) || [];
      const isOpen = q || !collapsed.has(p.pid);
      const match = q && (p.name.toLowerCase().includes(q) || String(p.pid) === q);
      const state = STATES[p.state] || p.state;
      rows.push(`<tr class="${isKernelThread(p) ? "kthread" : ""} ${match ? "match" : ""}">
        <td class="pid">${p.pid}</td>
        <td class="name"><span class="indent" style="width:${depth * 16}px"></span>${kids.length
          ? `<button class="twisty" data-pid="${p.pid}" aria-label="${isOpen ? "Collapse" : "Expand"} ${esc(p.name)}">${isOpen ? "▾" : "▸"}</button>`
          : `<span class="twisty"></span>`}${esc(p.name)}</td>
        <td><span class="st ${p.state === "Z" ? "z" : p.state === "R" ? "r" : ""}"><b>${esc(p.state)}</b>${esc(state)}</span></td>
        <td class="num">${p.uid}</td>
        <td class="num">${p.threads}</td>
        <td class="num">${p.rss_kb ? fmtKB(p.rss_kb) : "–"}</td>
      </tr>`);
      if (isOpen) walk(p.pid, depth + 1);
    }
  };
  walk(0, 0);
  $("#tree").innerHTML = rows.join("") || `<tr><td colspan="6">No processes match “${esc(q)}”.</td></tr>`;
  $("#treeHint").textContent = q
    ? `Matches for “${q}” with their parent processes.`
    : `${fmtInt(procs.length)} processes${showKernel ? "" : " (kernel threads hidden)"}. The tree is rebuilt from each process's parent PID.`;
}

$("#tree").addEventListener("click", (e) => {
  const b = e.target.closest(".twisty[data-pid]");
  if (!b) return;
  const pid = Number(b.dataset.pid);
  collapsed.has(pid) ? collapsed.delete(pid) : collapsed.add(pid);
  renderTree();
});
$("#search").addEventListener("input", () => snapshot && renderTree());
$("#showKernel").addEventListener("change", () => snapshot && renderTree());

refresh();
