// CRC: crc-VariableBrowser.md
// CRC: crc-HTTPEndpoint.md (R58, R63-R77)
package server

const variableBrowserHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Variable Browser</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: system-ui, -apple-system, sans-serif; padding: 16px; background: #fafafa; color: #333; }
h1 { font-size: 1.2em; font-weight: 600; margin-bottom: 12px; }

/* Toolbar */
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; padding: 8px 12px; background: #fff; border: 1px solid #ddd; border-radius: 6px; }
.toolbar label { font-size: 0.85em; cursor: pointer; display: flex; align-items: center; gap: 4px; }
.toolbar select { font-size: 0.85em; padding: 2px 4px; }
.toolbar button { font-size: 0.85em; padding: 4px 12px; border: 1px solid #ccc; border-radius: 4px; background: #fff; cursor: pointer; }
.toolbar button:hover { background: #f0f0f0; }
.toolbar .spacer { flex: 1; }

/* Column picker */
.col-picker { position: relative; }
.col-picker-menu { display: none; position: absolute; right: 0; top: 100%; background: #fff; border: 1px solid #ddd; border-radius: 4px; padding: 8px; z-index: 10; box-shadow: 0 2px 8px rgba(0,0,0,0.12); min-width: 160px; }
.col-picker-menu.open { display: block; }
.col-picker-menu label { display: block; padding: 3px 0; font-size: 0.85em; }

/* Table */
.table-wrap { overflow-x: auto; background: #fff; border: 1px solid #ddd; border-radius: 6px; }
table { border-collapse: collapse; font-size: 0.85em; width: 100%; }
thead { background: #f5f5f5; position: sticky; top: 0; z-index: 5; }
th { text-align: left; padding: 6px 10px; font-weight: 600; border-bottom: 2px solid #ddd; white-space: nowrap; user-select: none; }
th.sortable { cursor: pointer; }
th.sortable:hover { background: #eee; }
th .sort-arrow { margin-left: 4px; font-size: 0.7em; }
td { padding: 4px 10px; border-bottom: 1px solid #eee; white-space: nowrap; vertical-align: top; }
tr:hover { background: #f8f8ff; }

/* Column styles */
.col-diags { width: 24px; max-width: 24px; padding: 4px 2px !important; }
.col-id { color: #888; font-size: 0.9em; width: 40px; max-width: 40px; text-align: right; padding-right: 8px !important; }
.col-path { font-weight: 500; }
.col-type { color: #0066cc; font-weight: 600; }
.col-value { color: #228b22; font-family: monospace; font-size: 0.95em; max-width: 300px; overflow: hidden; text-overflow: ellipsis; }
.col-gotype { color: #666; font-family: monospace; font-size: 0.9em; }
.col-time { font-family: monospace; font-size: 0.9em; color: #555; }
.col-maxtime { font-family: monospace; font-size: 0.9em; color: #555; }
.col-error { color: #cc0000; font-family: monospace; font-size: 0.9em; }
.col-access { font-family: monospace; font-size: 0.9em; color: #666; }
.col-active { text-align: center; }
.col-props { font-size: 0.85em; color: #888; max-width: 200px; overflow: hidden; text-overflow: ellipsis; }
.col-spacer { width: 100%; }
td.has-error { background: #fff0f0; }

/* Diag toggle */
.diag-btn { background: none; border: 1px solid #ccc; border-radius: 3px; cursor: pointer; font-size: 0.75em; padding: 1px 5px; color: #666; }
.diag-btn:hover { background: #eee; }
.diag-btn.open { background: #e8e8e8; }

/* Diag sub-row */
tr.diag-row td { padding: 2px 10px 6px 40px; background: #f9f9f5; border-bottom: 1px solid #eee; }
tr.diag-row ul { list-style: none; padding: 0; margin: 0; }
tr.diag-row li { font-family: monospace; font-size: 0.85em; color: #555; padding: 1px 0; }
tr.diag-row li::before { content: "- "; color: #999; }

/* Tree indent */
.tree-toggle { display: inline-block; width: 16px; text-align: center; cursor: pointer; color: #888; font-size: 0.8em; user-select: none; }
.tree-toggle:hover { color: #333; }
.tree-leaf { display: inline-block; width: 16px; }

/* Status bar */
.status { font-size: 0.8em; color: #888; margin-top: 8px; }
</style>
</head>
<body>

<h1>Variable Browser</h1>

<div class="toolbar">
  <label><input type="radio" name="view" value="flat" checked> Flat</label>
  <label><input type="radio" name="view" value="tree"> Tree</label>
  <span class="spacer"></span>
  <button id="refreshBtn">Refresh</button>
  <label><input type="checkbox" id="pollToggle"> Poll</label>
  <select id="pollInterval">
    <option value="1000">1s</option>
    <option value="2000" selected>2s</option>
    <option value="5000">5s</option>
  </select>
  <span class="spacer"></span>
  <div class="col-picker">
    <button class="col-picker-btn" id="colPickerBtn">Columns &#9662;</button>
    <div class="col-picker-menu" id="colPickerMenu"></div>
  </div>
</div>

<div class="table-wrap">
  <table>
    <thead><tr id="headerRow"></tr></thead>
    <tbody id="tableBody"></tbody>
  </table>
</div>

<div class="status" id="status"></div>

<script>
(function() {
  'use strict';

  // Column definitions
  // R70, R71: default visible/hidden columns
  const COLUMNS = [
    { key: 'diags',   label: '',         visible: true,  sortable: false, alwaysVisible: true },
    { key: 'id',      label: 'ID',       visible: true,  sortable: true },
    { key: 'path',    label: 'Path',     visible: true,  sortable: true },
    { key: 'type',    label: 'Type',     visible: true,  sortable: true },
    { key: 'goType',  label: 'GoType',   visible: false, sortable: true },
    { key: 'value',   label: 'Value',    visible: true,  sortable: false },
    { key: 'time',    label: 'Time',     visible: true,  sortable: true, numeric: true },
    { key: 'maxTime', label: 'Max Time', visible: false, sortable: true, numeric: true },
    { key: 'error',   label: 'Error',    visible: true,  sortable: true },
    { key: 'access',  label: 'Access',   visible: false, sortable: true },
    { key: 'active',  label: 'Active',   visible: false, sortable: true },
    { key: 'props',   label: 'Props',    visible: false, sortable: false },
  ];

  let variables = [];
  let viewMode = 'flat';      // R64, R65, R66
  let sortCol = null;
  let sortDir = 'asc';
  let pollTimer = null;
  let expandedDiags = new Set();
  let collapsedNodes = new Set();

  // Extract session ID from URL path
  const pathParts = location.pathname.split('/').filter(Boolean);
  const sessionId = pathParts[0] || '';

  // --- Data fetching ---
  // R57, R67
  async function fetchVariables() {
    const url = '/' + sessionId + '/variables.json';
    try {
      const resp = await fetch(url);
      if (!resp.ok) throw new Error(resp.status + ' ' + resp.statusText);
      variables = await resp.json();
      render();
      document.getElementById('status').textContent =
        variables.length + ' variables loaded at ' + new Date().toLocaleTimeString();
    } catch (e) {
      document.getElementById('status').textContent = 'Error: ' + e.message;
    }
  }

  // --- Rendering ---
  function render() {
    renderHeader();
    renderBody();
  }

  // R63: table with fixed header
  function renderHeader() {
    const tr = document.getElementById('headerRow');
    tr.innerHTML = '';
    for (const col of COLUMNS) {
      if (!col.visible) continue;
      const th = document.createElement('th');
      th.textContent = col.label;
      th.className = 'col-' + col.key;
      th.dataset.col = col.key;
      // R72: sortable headers in flat mode
      if (col.sortable && viewMode === 'flat') {
        th.classList.add('sortable');
        th.onclick = () => toggleSort(col.key);
        if (sortCol === col.key) {
          const arrow = document.createElement('span');
          arrow.className = 'sort-arrow';
          arrow.textContent = sortDir === 'asc' ? '\u25B2' : '\u25BC';
          th.appendChild(arrow);
        }
      }
      tr.appendChild(th);
    }
    // Spacer column absorbs remaining width
    const spacer = document.createElement('th');
    spacer.className = 'col-spacer';
    tr.appendChild(spacer);
  }

  function renderBody() {
    const tbody = document.getElementById('tableBody');
    tbody.innerHTML = '';
    const rows = viewMode === 'tree' ? buildTreeRows() : buildFlatRows();
    for (const row of rows) {
      tbody.appendChild(createDataRow(row));
      // R73, R74: diag sub-rows
      if (expandedDiags.has(row.v.id) && row.v.diags && row.v.diags.length > 0) {
        tbody.appendChild(createDiagRow(row.v));
      }
    }
  }

  // R64: tree mode with indentation and expand/collapse
  function buildTreeRows() {
    const byParent = new Map();
    for (const v of variables) {
      if (!byParent.has(v.parentId)) byParent.set(v.parentId, []);
      byParent.get(v.parentId).push(v);
    }

    const rows = [];
    function walk(parentId, depth) {
      const children = byParent.get(parentId) || [];
      for (const v of children) {
        const hasChildren = byParent.has(v.id) && byParent.get(v.id).length > 0;
        const collapsed = collapsedNodes.has(v.id);
        rows.push({ v, depth, hasChildren, collapsed });
        if (hasChildren && !collapsed) {
          walk(v.id, depth + 1);
        }
      }
    }
    walk(0, 0);
    return rows;
  }

  // R65: flat mode
  function buildFlatRows() {
    const sorted = [...variables];
    // R72: sortable columns
    if (sortCol) {
      const colDef = COLUMNS.find(c => c.key === sortCol);
      const numeric = colDef && colDef.numeric;
      const dir = sortDir === 'asc' ? 1 : -1;
      sorted.sort((a, b) => {
        let va = getCellValue(a, sortCol);
        let vb = getCellValue(b, sortCol);
        if (numeric) {
          va = parseFloat(va) || 0;
          vb = parseFloat(vb) || 0;
        }
        if (va < vb) return -dir;
        if (va > vb) return dir;
        return 0;
      });
    }
    return sorted.map(v => ({ v, depth: 0, hasChildren: false, collapsed: false }));
  }

  function getCellValue(v, colKey) {
    switch (colKey) {
      case 'id': return v.id;
      case 'path': return v.path || '';
      case 'type': return v.type || '';
      case 'time': return v.computeTime || '';
      case 'maxTime': return v.maxComputeTime || '';
      case 'error': return v.error || '';
      case 'goType': return v.goType || '';
      case 'access': return v.access || '';
      case 'active': return v.active ? '1' : '0';
      default: return '';
    }
  }

  function createDataRow(row) {
    const { v, depth, hasChildren, collapsed } = row;
    const tr = document.createElement('tr');
    tr.dataset.id = v.id;

    for (const col of COLUMNS) {
      if (!col.visible) continue;
      const td = document.createElement('td');
      td.className = 'col-' + col.key;

      switch (col.key) {
        case 'diags':
          // R73: diag toggle button
          if (v.diags && v.diags.length > 0) {
            const btn = document.createElement('button');
            btn.className = 'diag-btn' + (expandedDiags.has(v.id) ? ' open' : '');
            btn.textContent = expandedDiags.has(v.id) ? 'D\u25BC' : 'D\u25B6';
            btn.onclick = () => { toggleDiag(v.id); };
            td.appendChild(btn);
          }
          break;

        case 'id':
          td.textContent = v.id;
          break;

        case 'path': {
          // R64: tree indentation
          if (viewMode === 'tree') {
            td.style.paddingLeft = (10 + depth * 20) + 'px';
            if (hasChildren) {
              const toggle = document.createElement('span');
              toggle.className = 'tree-toggle';
              toggle.textContent = collapsed ? '\u25B6' : '\u25BC';
              toggle.onclick = () => { toggleNode(v.id); };
              td.appendChild(toggle);
            } else {
              const spacer = document.createElement('span');
              spacer.className = 'tree-leaf';
              td.appendChild(spacer);
            }
          }
          if (viewMode === 'tree' && hasChildren) {
            const label = document.createElement('span');
            label.textContent = v.path || '(root)';
            label.style.cursor = 'pointer';
            label.onclick = () => { toggleNode(v.id); };
            td.appendChild(label);
          } else {
            td.appendChild(document.createTextNode(v.path || '(root)'));
          }
          break;
        }

        case 'type':
          td.textContent = v.type || '';
          break;

        case 'value': {
          // R75: truncated with tooltip
          const full = v.value != null ? JSON.stringify(v.value) : '';
          const display = full.length > 100 ? full.slice(0, 100) + '\u2026' : full;
          td.textContent = display;
          if (full.length > 100) td.title = full;
          break;
        }

        case 'time':
          td.textContent = v.computeTime || '';
          break;

        case 'maxTime':
          td.textContent = v.maxComputeTime || '';
          break;

        case 'error':
          // R76: red highlight
          td.textContent = v.error || '';
          if (v.error) td.classList.add('has-error');
          break;

        case 'goType':
          td.textContent = v.goType || '';
          break;

        case 'access':
          td.textContent = v.access || '';
          break;

        case 'active':
          td.textContent = v.active ? '\u2713' : '\u2717';
          break;

        case 'props': {
          const parts = [];
          if (v.properties) {
            for (const [k, val] of Object.entries(v.properties)) {
              if (k !== 'type' && k !== 'path' && k !== 'access') {
                parts.push(k + '=' + val);
              }
            }
          }
          td.textContent = parts.join(', ');
          if (parts.length) td.title = parts.join('\n');
          break;
        }
      }
      tr.appendChild(td);
    }
    // Spacer cell to match header
    const spacerTd = document.createElement('td');
    spacerTd.className = 'col-spacer';
    tr.appendChild(spacerTd);
    return tr;
  }

  // R74: diagnostic sub-row
  function createDiagRow(v) {
    const tr = document.createElement('tr');
    tr.className = 'diag-row';
    const td = document.createElement('td');
    const visibleCount = COLUMNS.filter(c => c.visible).length;
    td.colSpan = visibleCount + 1; // +1 for spacer column
    const ul = document.createElement('ul');
    for (const msg of v.diags) {
      const li = document.createElement('li');
      li.textContent = msg;
      ul.appendChild(li);
    }
    td.appendChild(ul);
    tr.appendChild(td);
    return tr;
  }

  // --- Interactions ---

  function toggleSetEntry(set, key) {
    if (set.has(key)) { set.delete(key); } else { set.add(key); }
  }

  function toggleSort(colKey) {
    if (sortCol === colKey) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortCol = colKey;
      const col = COLUMNS.find(c => c.key === colKey);
      sortDir = col && col.numeric ? 'desc' : 'asc';
    }
    render();
  }

  function toggleDiag(id) {
    toggleSetEntry(expandedDiags, id);
    render();
  }

  function toggleNode(id) {
    toggleSetEntry(collapsedNodes, id);
    render();
  }

  // R66: view mode toggle
  document.querySelectorAll('input[name="view"]').forEach(radio => {
    radio.addEventListener('change', () => {
      viewMode = radio.value;
      sortCol = null;
      render();
    });
  });

  // R67: refresh button
  document.getElementById('refreshBtn').addEventListener('click', fetchVariables);

  // R68: poll toggle
  const pollToggle = document.getElementById('pollToggle');
  const pollInterval = document.getElementById('pollInterval');

  function updatePolling() {
    if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
    if (pollToggle.checked) {
      const ms = parseInt(pollInterval.value, 10);
      pollTimer = setInterval(fetchVariables, ms);
    }
  }
  pollToggle.addEventListener('change', updatePolling);
  pollInterval.addEventListener('change', updatePolling);

  // R69: column picker
  function buildColumnPicker() {
    const menu = document.getElementById('colPickerMenu');
    menu.innerHTML = '';
    for (const col of COLUMNS) {
      if (col.alwaysVisible) continue;
      const label = document.createElement('label');
      const cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.checked = col.visible;
      cb.addEventListener('change', () => {
        col.visible = cb.checked;
        render();
      });
      label.appendChild(cb);
      label.appendChild(document.createTextNode(' ' + col.label));
      menu.appendChild(label);
    }
  }

  document.getElementById('colPickerBtn').addEventListener('click', (e) => {
    e.stopPropagation();
    document.getElementById('colPickerMenu').classList.toggle('open');
  });
  document.addEventListener('click', () => {
    document.getElementById('colPickerMenu').classList.remove('open');
  });
  document.getElementById('colPickerMenu').addEventListener('click', (e) => {
    e.stopPropagation();
  });

  // --- Init ---
  buildColumnPicker();
  fetchVariables();
})();
</script>
</body>
</html>`
