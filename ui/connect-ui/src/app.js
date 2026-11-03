(() => {
  const state = {
    self: null,
    workspaces: [],
    workspace: null,
    folders: [],
    items: [],
    folder: "all",
    selected: null,
    typeFilter: "all",
    ageFilter: "all",
    search: "",
    sort: "recent:desc",
    view: "grid",
    invitePolling: null,
  };

  const el = (id) => document.getElementById(id);
  const iconPaths = {
    "arrow-left": ["M19 12H5", "m12 19-7-7 7-7"],
    more: ["M5 12h.01M12 12h.01M19 12h.01"],
    trash: ["M4 7h16", "M10 11v6M14 11v6", "m9 7 1-2h4l1 2", "M6 7l1 13h10l1-13"],
    text: ["M5 5h14M12 5v14", "M8 19h8"],
    "folder-plus": ["M3 7.5A2.5 2.5 0 0 1 5.5 5H10l2 2h6.5A2.5 2.5 0 0 1 21 9.5v8a2.5 2.5 0 0 1-2.5 2.5h-13A2.5 2.5 0 0 1 3 17.5z", "M12 11v5M9.5 13.5h5"],
    search: ["m20 20-4.4-4.4", "a7 7 0 1 1-9.9-9.9 7 7 0 0 1 9.9 9.9"],
    grid: ["M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z"],
    list: ["M8 6h12M8 12h12M8 18h12", "M4 6h.01M4 12h.01M4 18h.01"],
    sparkle: ["m12 3 1.6 5.4L19 10l-5.4 1.6L12 17l-1.6-5.4L5 10l5.4-1.6z", "m19 16 .6 2.1L22 19l-2.4.9L19 22l-.6-2.1L16 19l2.4-.9z"],
    device: ["M5 4h14a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z", "M8 21h8M12 17v4"],
    file: ["M6 3h8l4 4v14H6z", "M14 3v5h5", "M9 14h6M9 17h5"],
    code: ["m8 9-3 3 3 3M16 9l3 3-3 3M14 6l-4 12"],
    image: ["M4 5h16v14H4z", "m7 16 3.5-4 2.5 3 2-2 3 3", "M8 9h.01"],
    archive: ["M4 7h16v13H4z", "M3 4h18v3H3z", "M10 11h4M10 15h4"],
    folder: ["M3 7.5A2.5 0 0 1 5.5 5H10l2 2h6.5A2.5 2.5 0 0 1 21 9.5v8a2.5 2.5 0 0 1-2.5 2.5h-13A2.5 2.5 0 0 1 3 17.5z"],
    upload: ["M12 15V3", "m8 7 4-4 4 4", "M4 20h16"],
    download: ["M12 3v12", "m8 11 4 4 4-4", "M4 20h16"],
    copy: ["M9 8h9v12H9z", "M6 16H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"],
  };

  function icon(name) {
    const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("fill", "none");
    svg.setAttribute("stroke", "currentColor");
    svg.setAttribute("stroke-width", "1.9");
    svg.setAttribute("stroke-linecap", "round");
    svg.setAttribute("stroke-linejoin", "round");
    svg.setAttribute("aria-hidden", "true");
    svg.className = "ui-icon";
    for (const d of iconPaths[name] || iconPaths.file) {
      const path = document.createElementNS(svg.namespaceURI, "path");
      path.setAttribute("d", d);
      svg.append(path);
    }
    return svg;
  }

  function hydrateIcons() {
    document.querySelectorAll("[data-icon]").forEach((node) => node.replaceChildren(icon(node.dataset.icon)));
  }

  async function api(path, options) {
    const base = (window.CONNECT && window.CONNECT.apiBase) || "";
    const response = await fetch(base + path, options);
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || response.statusText);
    }
    return response.json();
  }

  function json(method, body) {
    return { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) };
  }

  function workspaceID() {
    return state.workspace && state.workspace.id;
  }

  async function load() {
    // Wait for port detection before making API calls
    if (window.CONNECT && window.CONNECT.portReady) await window.CONNECT.portReady;
    try {
      state.self = await api("/api/self");
    } catch (e) {
      el("self-name").textContent = "Connection error";
      el("self-address").textContent = "Could not reach connectd service";
      throw e;
    }
    el("self-name").textContent = state.self.name;
    el("self-address").textContent = `${state.self.ip}:${state.self.port}`;
    try {
      state.workspaces = await api("/api/workspaces");
    } catch (e) {
      state.workspaces = [];
    }
    renderDashboard();
  }

  function renderDashboard() {
    const grid = el("connections-grid");
    grid.replaceChildren();
    el("connections-empty").hidden = state.workspaces.length > 0;
    el("connections-caption").textContent = state.workspaces.length
      ? `${state.workspaces.length} paired ${state.workspaces.length === 1 ? "device" : "devices"}`
      : "Pair a second computer to create a shared workspace.";
    for (const workspace of state.workspaces) grid.append(connectionCard(workspace));
  }

  function connectionCard(workspace) {
    const card = document.createElement("article");
    card.className = `connection-card ${statusClass(workspace.status)}`;
    const open = document.createElement("a");
    open.className = "connection-card-main";
    open.href = `#connection/${encodeURIComponent(workspace.id)}`;
    const heading = document.createElement("div");
    heading.className = "connection-card-heading";
    const mark = document.createElement("div");
    mark.className = "device-mark";
    setDeviceMark(mark, workspace.name);
    const identity = document.createElement("div");
    const name = document.createElement("h3");
    name.textContent = workspace.name;
    const description = document.createElement("p");
    description.textContent = "Shared paired workspace";
    identity.append(name, description);
    heading.append(mark, identity);
    const status = document.createElement("span");
    status.className = `status-pill ${statusClass(workspace.status)}`;
    status.textContent = statusLabel(workspace.status);
    const metrics = document.createElement("dl");
    metrics.className = "connection-metrics";
    metrics.append(metric("Artifacts", workspace.items || 0));
    metrics.append(metric("Stale", workspace.stale_items || 0, workspace.stale_items > 0 ? "attention" : ""));
    open.append(heading, status, metrics);
    const actions = document.createElement("div");
    actions.className = "connection-card-actions";
    actions.append(actionButton("Remove", "button button-quiet button-small", () => removeWorkspace(workspace.id), "trash"));
    card.append(open, actions);
    return card;
  }

  function metric(label, value, extraClass = "") {
    const item = document.createElement("div");
    if (extraClass) item.className = extraClass;
    const dd = document.createElement("dd");
    dd.textContent = String(value);
    const dt = document.createElement("dt");
    dt.textContent = label;
    item.append(dd, dt);
    return item;
  }

  async function openWorkspace(id) {
    state.workspace = await api(`/api/workspaces/${encodeURIComponent(id)}`);
    state.folder = "all";
    state.selected = null;
    state.typeFilter = "all";
    state.ageFilter = "all";
    state.search = "";
    el("workspace-search").value = "";
    el("connections-screen").hidden = true;
    el("details-screen").hidden = false;
    renderDetailHeader();
    await refresh();
  }

  function renderDetailHeader() {
    if (!state.workspace) return;
    setDeviceMark(el("detail-device-mark"), state.workspace.name);
    el("detail-connection-name").textContent = state.workspace.name;
    const status = el("detail-status");
    status.className = `status-pill ${statusClass(state.workspace.status)}`;
    status.textContent = statusLabel(state.workspace.status);
    el("detail-connection-address").textContent = "Shared paired workspace";
  }

  async function refresh() {
    const id = workspaceID();
    if (!id) return;
    const [workspace, folders, items] = await Promise.all([
      api(`/api/workspaces/${encodeURIComponent(id)}`),
      api(`/api/workspaces/${encodeURIComponent(id)}/folders`),
      api(`/api/workspaces/${encodeURIComponent(id)}/artifacts?folder=${encodeURIComponent(state.folder)}`),
    ]);
    state.workspace = workspace;
    state.folders = folders;
    state.items = items;
    if (state.selected && !state.items.some((item) => item.id === state.selected.id)) state.selected = null;
    renderDetailHeader();
    renderWorkspace();
  }

  function renderWorkspace() {
    renderFolders();
    renderWorkspaceControls();
    renderItems();
    renderInspector();
  }

  function renderFolders() {
    const list = el("folder-list");
    list.replaceChildren();
    list.append(folderButton({ id: "all", name: "All artifacts", item_count: state.workspace ? state.workspace.items : state.items.length, stale_count: state.workspace ? state.workspace.stale_items : 0 }));
    for (const folder of state.folders) list.append(folderButton(folder));
  }

  function folderButton(folder) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = `folder-button ${state.folder === folder.id ? "active" : ""}`;
    button.setAttribute("aria-current", state.folder === folder.id ? "page" : "false");
    const label = document.createElement("span");
    label.className = "folder-button-label";
    const symbol = document.createElement("span");
    symbol.className = "folder-symbol";
    symbol.append(icon(folder.id === "all" ? "grid" : "folder"));
    const name = document.createElement("strong");
    name.textContent = folder.name;
    label.append(symbol, name);
    const count = document.createElement("span");
    count.className = folder.stale_count > 0 ? "folder-count attention" : "folder-count";
    count.textContent = String(folder.item_count || 0);
    button.append(label, count);
    button.addEventListener("dragover", (event) => {
      event.preventDefault();
      event.dataTransfer.dropEffect = "move";
      button.classList.add("drop-target");
    });
    button.addEventListener("dragleave", () => button.classList.remove("drop-target"));
    button.addEventListener("drop", async (event) => {
      event.preventDefault();
      button.classList.remove("drop-target");
      const itemID = event.dataTransfer.getData("application/x-connect-artifact");
      if (itemID) await moveItemToFolder(itemID, folder.id === "all" ? "" : folder.id);
    });
    button.addEventListener("click", async () => {
      if (state.folder === folder.id) return;
      state.folder = folder.id;
      state.selected = null;
      await refresh();
    });
    return button;
  }

  function renderWorkspaceControls() {
    const selectedFolder = state.folders.find((folder) => folder.id === state.folder);
    el("workspace-title").textContent = selectedFolder ? selectedFolder.name : "All artifacts";
    el("sort-select").value = state.sort;
    document.querySelectorAll("[data-type-filter]").forEach((button) => {
      button.classList.toggle("active", button.dataset.typeFilter === state.typeFilter);
      button.setAttribute("aria-pressed", String(button.dataset.typeFilter === state.typeFilter));
    });
    document.querySelectorAll("[data-age-filter]").forEach((button) => {
      button.classList.toggle("active", button.dataset.ageFilter === state.ageFilter);
      button.setAttribute("aria-pressed", String(button.dataset.ageFilter === state.ageFilter));
    });
    document.querySelectorAll("[data-view-mode]").forEach((button) => {
      button.classList.toggle("active", button.dataset.viewMode === state.view);
      button.setAttribute("aria-pressed", String(button.dataset.viewMode === state.view));
    });
    el("workspace-grid").classList.toggle("list-mode", state.view === "list");
  }

  function visibleItems() {
    const needle = state.search.trim().toLowerCase();
    const folder = state.folders.find((candidate) => candidate.id === state.folder);
    const items = state.items.filter((item) => {
      if (state.typeFilter !== "all" && item.type !== state.typeFilter) return false;
      if (state.ageFilter === "attention" && ageClass(item) === "fresh") return false;
      if (!needle) return true;
      return [item.title, item.filename, item.content, item.type, fileKind(item.filename || item.title), folder ? folder.name : ""]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(needle));
    });
    return sortItems(items);
  }

  function sortItems(items) {
    const [sortBy, direction] = state.sort.split(":");
    const sign = direction === "asc" ? 1 : -1;
    return [...items].sort((a, b) => {
      if (sortBy === "name") return sign * textCompare(a.title || a.filename, b.title || b.filename);
      if (sortBy === "type") return sign * textCompare(itemKind(a), itemKind(b)) || textCompare(a.title || a.filename, b.title || b.filename);
      if (sortBy === "stale") return sign * (daysSince(a.updated_at || a.created_at) - daysSince(b.updated_at || b.created_at));
      if (sortBy === "added") return sign * (dateValue(a.created_at) - dateValue(b.created_at));
      return sign * (dateValue(a.updated_at || a.created_at) - dateValue(b.updated_at || b.created_at));
    });
  }

  function renderItems() {
    const grid = el("workspace-grid");
    grid.querySelectorAll(".artifact-card").forEach((node) => node.remove());
    const items = visibleItems();
    const empty = el("workspace-empty");
    const hasArtifacts = state.items.length > 0;
    empty.hidden = items.length > 0;
    el("workspace-empty-title").textContent = hasArtifacts ? "No matching artifacts" : "This workspace is ready";
    el("workspace-empty-description").textContent = hasArtifacts
      ? "Try a different search, filter, sort or folder."
      : "Add files, create a text artifact or add a folder to start organizing this connection.";
    el("empty-create-text-button").hidden = hasArtifacts;
    el("empty-add-files-button").hidden = hasArtifacts;
    el("clear-workspace-filters-button").hidden = !hasArtifacts;
    for (const item of items) grid.append(itemCard(item));
    el("workspace-result-count").textContent = resultCount(items.length);
  }

  function itemCard(item) {
    const card = document.createElement("button");
    card.type = "button";
    card.draggable = true;
    const age = ageClass(item);
    card.className = `artifact-card ${item.type} ${age} ${state.selected && state.selected.id === item.id ? "selected" : ""}`;
    card.setAttribute("aria-pressed", String(state.selected && state.selected.id === item.id));
    const top = document.createElement("div");
    top.className = "artifact-card-top";
    const kind = document.createElement("span");
    kind.className = `artifact-kind ${item.type}`;
    kind.append(icon(itemIconName(item)));
    const kindLabel = document.createElement("span");
    kindLabel.textContent = itemKind(item);
    kind.append(kindLabel);
    const ageNode = document.createElement("span");
    ageNode.className = `age-pill ${age}`;
    ageNode.textContent = ageLabel(item);
    top.append(kind, ageNode);
    const title = document.createElement("strong");
    title.className = "artifact-title";
    title.textContent = item.title || item.filename || "Untitled artifact";
    const preview = document.createElement("p");
    preview.className = "artifact-preview";
    preview.textContent = item.type === "text" ? textPreview(item.content) : fileSummary(item);
    const footer = document.createElement("span");
    footer.className = "artifact-footer";
    footer.textContent = `${folderName(item.folder_id)} · ${activityLabel(item)}`;
    card.append(top, title, preview, footer);
    card.addEventListener("click", () => {
      state.selected = item;
      renderItems();
      renderInspector();
    });
    card.addEventListener("dragstart", (event) => {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("application/x-connect-artifact", item.id);
      event.dataTransfer.setData("text/plain", item.title || item.filename || "artifact");
      card.classList.add("dragging");
    });
    card.addEventListener("dragend", () => card.classList.remove("dragging"));
    return card;
  }

  function renderInspector() {
    const empty = el("inspector-empty");
    const details = el("inspector-details");
    if (!state.selected) {
      empty.hidden = false;
      details.hidden = true;
      return;
    }
    const item = state.selected;
    empty.hidden = true;
    details.hidden = false;
    el("inspector-kind").replaceChildren(icon(itemIconName(item)), document.createTextNode(itemKind(item)));
    el("inspector-title").textContent = item.title || item.filename || "Untitled artifact";
    const content = el("inspector-content");
    content.hidden = item.type !== "text";
    content.textContent = item.type === "text" ? item.content || "" : "";
    renderInspectorMeta(item);
    const actions = el("inspector-actions");
    actions.replaceChildren();
    if (item.type === "text") actions.append(actionButton("Copy text", "button button-primary", () => copyTextItem(item), "copy"));
    if (item.type === "file") {
      const link = document.createElement("a");
      link.className = "button button-primary";
      link.href = `/api/workspaces/${encodeURIComponent(workspaceID())}/artifacts/${encodeURIComponent(item.id)}/file`;
      link.textContent = "Download file";
      link.prepend(icon("download"));
      actions.append(link);
    }
    actions.append(moveItemControl(item));
    actions.append(actionButton("Delete artifact", "button button-danger", () => deleteItem(item), "trash"));
  }

  function renderInspectorMeta(item) {
    const meta = el("inspector-meta");
    meta.replaceChildren();
    const rows = [
      ["Type", item.type === "text" ? "Text artifact" : fileTypeLabel(item)],
      ["Folder", folderName(item.folder_id)],
      ["Added", formatDate(item.created_at)],
      ["Last activity", activityLabel(item)],
      ["Age", ageLabel(item)],
      ["Status", item.status || "ready"],
    ];
    if (item.type === "file") rows.splice(1, 0, ["Size", formatBytes(item.size)]);
    for (const [label, value] of rows) {
      const dt = document.createElement("dt");
      dt.textContent = label;
      const dd = document.createElement("dd");
      dd.textContent = value;
      meta.append(dt, dd);
    }
  }

  function moveItemControl(item) {
    const control = document.createElement("div");
    control.className = "move-item-control";
    const select = document.createElement("select");
    select.setAttribute("aria-label", "Move artifact to folder");
    const unfiled = document.createElement("option");
    unfiled.value = "";
    unfiled.textContent = "Unfiled";
    select.append(unfiled);
    for (const folder of state.folders) {
      const option = document.createElement("option");
      option.value = folder.id;
      option.textContent = folder.name;
      select.append(option);
    }
    select.value = item.folder_id || "";
    const button = actionButton("Move", "button button-secondary", async () => moveItemToFolder(item.id, select.value), "folder");
    control.append(select, button);
    return control;
  }

  async function moveItemToFolder(itemID, folderID) {
    const current = state.items.find((item) => item.id === itemID);
    if (current && (current.folder_id || "") === folderID) return;
    await api(`/api/workspaces/${encodeURIComponent(workspaceID())}/artifacts/${encodeURIComponent(itemID)}`, json("PATCH", { folder_id: folderID }));
    showToast(`Moved to ${folderID ? folderName(folderID) : "Unfiled"}`);
    await refresh();
  }

  async function deleteItem(item) {
    await api(`/api/workspaces/${encodeURIComponent(workspaceID())}/artifacts/${encodeURIComponent(item.id)}`, { method: "DELETE" });
    state.selected = null;
    showToast("Artifact deleted");
    await refresh();
  }

  async function copyTextItem(item) {
    await copyToClipboard(item.content || "");
    showToast("Text copied");
  }

  async function submitText(event) {
    event.preventDefault();
    clearError("text-form-error");
    const content = el("text-content").value.trim();
    if (!content) return;
    try {
      await api(`/api/workspaces/${encodeURIComponent(workspaceID())}/artifacts/text`, json("POST", { title: el("text-title").value.trim(), content, folder_id: state.folder === "all" ? "" : state.folder }));
      el("text-dialog").close();
      event.target.reset();
      showToast("Text artifact created");
      await refresh();
    } catch (error) {
      showError("text-form-error", error.message);
    }
  }

  async function submitFolder(event) {
    event.preventDefault();
    clearError("folder-form-error");
    try {
      await api(`/api/workspaces/${encodeURIComponent(workspaceID())}/folders`, json("POST", { name: el("folder-name").value.trim() }));
      el("folder-dialog").close();
      event.target.reset();
      showToast("Folder created");
      await refresh();
    } catch (error) {
      showError("folder-form-error", error.message);
    }
  }

  async function upload() {
    const files = Array.from(el("file-upload-input").files || []);
    el("file-upload-input").value = "";
    if (!files.length) return;
    for (const file of files) {
      const form = new FormData();
      form.append("file", file);
      if (state.folder !== "all") form.append("folder_id", state.folder);
      await api(`/api/workspaces/${encodeURIComponent(workspaceID())}/artifacts/files`, { method: "POST", body: form });
    }
    showToast(`${files.length} ${files.length === 1 ? "file" : "files"} added`);
    await refresh();
  }

  async function removeWorkspace(id) {
    try {
      await api(`/api/workspaces/${encodeURIComponent(id)}`, { method: "DELETE" });
      if (state.workspace && state.workspace.id === id) location.hash = "";
      await load();
      showToast("Workspace removed");
    } catch (e) {
      showToast("Failed to remove workspace: " + e.message);
    }
  }

  function openTextDialog() {
    clearError("text-form-error");
    el("text-title").value = "";
    el("text-content").value = "";
    const folder = state.folders.find((candidate) => candidate.id === state.folder);
    el("text-folder-hint").textContent = folder ? `This text will be saved in ${folder.name}.` : "This text will be saved outside folders.";
    dialog("text-dialog");
  }

  function openPairDialog() {
    clearInviteError();
    clearPairError();
    dialog("connection-dialog");
  }

  function dialog(name) {
    const node = el(name);
    if (!node.open) node.showModal();
  }

  function showError(id, message) {
    const node = el(id);
    node.textContent = message;
    node.hidden = false;
  }

  function clearError(id) {
    const node = el(id);
    node.textContent = "";
    node.hidden = true;
  }

  function showInviteError(message) {
    showError("invite-form-error", message);
  }

  function clearInviteError() {
    clearError("invite-form-error");
  }

  function showPairError(message) {
    showError("pair-form-error", message);
  }

  function clearPairError() {
    clearError("pair-form-error");
  }

  async function copyInvitation() {
    const input = el("invite-url");
    input.focus();
    input.select();
    await copyToClipboard(input.value);
    el("copy-invite-button").textContent = "Copied";
    setTimeout(() => { el("copy-invite-button").textContent = "Copy"; }, 1500);
  }

  async function createInvitation() {
    const button = el("create-invite-button");
    button.disabled = true;
    button.textContent = "Creating...";
    clearInviteError();
    try {
      const invitation = await api("/api/invitations", { method: "POST" });
      const url = invitation.url;
      el("invite-url").value = url;
      el("invite-result").hidden = false;
      const subject = encodeURIComponent("Connect invitation");
      const body = encodeURIComponent(`Join my Connect shared workspace:\n${url}\n\nPaste this link into your local Connect app.`);
      el("email-invite-button").href = `mailto:?subject=${subject}&body=${body}`;
      await copyInvitation();
      showToast("Invitation created and copied");
      // Poll for new paired workspace
      const prevCount = state.workspaces.length;
      state.invitePolling = setInterval(async () => {
        try {
          const workspaces = await api("/api/workspaces");
          if (workspaces.length > prevCount) {
            clearInterval(state.invitePolling);
            state.invitePolling = null;
            el("connection-dialog").close();
            showToast("Device connected!");
            await load();
            location.hash = "#connection/" + encodeURIComponent(workspaces[workspaces.length - 1].id);
          }
        } catch (_) {}
      }, 2000);
    } catch (error) {
      showInviteError(`Could not create invitation: ${error.message}`);
    } finally {
      button.disabled = false;
      button.textContent = "Create invitation";
    }
  }

  async function shareInvitation() {
    const url = el("invite-url").value;
    if (!url) return;
    if (navigator.share) {
      try {
        await navigator.share({ title: "Connect invitation", text: "Join my Connect shared workspace", url });
        return;
      } catch (_) {}
    }
    await copyInvitation();
  }

  async function submitPair(event) {
    event.preventDefault();
    clearPairError();
    try {
      const workspace = await api("/api/workspaces/pair", json("POST", { url: el("pair-url").value }));
      el("connection-dialog").close();
      event.target.reset();
      await load();
      location.hash = `connection/${encodeURIComponent(workspace.id)}`;
      showToast("Device paired");
    } catch (error) {
      showPairError(error.message);
    }
  }

  function route() {
    const match = location.hash.match(/^#connection\/(.+)$/);
    if (match) {
      void openWorkspace(decodeURIComponent(match[1]));
      return;
    }
    state.workspace = null;
    state.selected = null;
    el("connections-screen").hidden = false;
    el("details-screen").hidden = true;
    void load();
  }

  function registerEvents() {
    el("back-to-connections").onclick = () => { location.hash = ""; };
    el("empty-connect-device-button").onclick = openPairDialog;
    el("connections-menu-button").onclick = openPairDialog;
    el("create-invite-button").onclick = () => void createInvitation();
    el("copy-invite-button").onclick = () => void copyInvitation();
    el("share-invite-button").onclick = () => void shareInvitation();
    el("create-text-button").onclick = openTextDialog;
    el("empty-create-text-button").onclick = openTextDialog;
    el("new-folder-button").onclick = () => { clearError("folder-form-error"); dialog("folder-dialog"); };
    el("add-files-button").onclick = () => el("file-upload-input").click();
    el("empty-add-files-button").onclick = () => el("file-upload-input").click();
    el("file-upload-input").onchange = () => void upload();
    el("remove-connection-button").onclick = () => { if (state.workspace) void removeWorkspace(state.workspace.id); };
    el("pair-form").onsubmit = (event) => void submitPair(event);
    el("text-create-form").onsubmit = (event) => void submitText(event);
    el("folder-create-form").onsubmit = (event) => void submitFolder(event);
    el("workspace-search").oninput = (event) => {
      state.search = event.target.value;
      state.selected = null;
      renderItems();
      renderInspector();
    };
    el("sort-select").onchange = (event) => {
      state.sort = event.target.value;
      renderWorkspaceControls();
      renderItems();
      renderInspector();
    };
    el("clear-workspace-filters-button").onclick = () => {
      state.search = "";
      state.typeFilter = "all";
      state.ageFilter = "all";
      state.selected = null;
      el("workspace-search").value = "";
      renderWorkspace();
    };
    document.querySelectorAll("[data-type-filter]").forEach((button) => {
      button.onclick = () => {
        state.typeFilter = button.dataset.typeFilter;
        state.selected = null;
        renderWorkspaceControls();
        renderItems();
        renderInspector();
      };
    });
    document.querySelectorAll("[data-age-filter]").forEach((button) => {
      button.onclick = () => {
        state.ageFilter = button.dataset.ageFilter;
        state.selected = null;
        renderWorkspaceControls();
        renderItems();
        renderInspector();
      };
    });
    document.querySelectorAll("[data-view-mode]").forEach((button) => {
      button.onclick = () => {
        state.view = button.dataset.viewMode;
        renderWorkspaceControls();
        renderItems();
      };
    });
    document.querySelectorAll("[data-close-dialog]").forEach((button) => {
      button.onclick = () => el(button.dataset.closeDialog).close();
    });
    el("connection-dialog").addEventListener("close", () => {
      if (state.invitePolling) {
        clearInterval(state.invitePolling);
        state.invitePolling = null;
      }
    });
    window.onhashchange = route;
  }

  function actionButton(label, className, onClick, iconName = "") {
    const button = document.createElement("button");
    button.type = "button";
    button.className = className;
    if (iconName) button.append(icon(iconName));
    button.append(document.createTextNode(label));
    button.addEventListener("click", onClick);
    return button;
  }

  async function copyToClipboard(value) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      try {
        await navigator.clipboard.writeText(value);
        return;
      } catch (_) {}
    }
    const field = document.createElement("textarea");
    field.value = value;
    field.setAttribute("readonly", "");
    field.style.cssText = "position:fixed;opacity:0;pointer-events:none";
    document.body.append(field);
    field.select();
    document.execCommand("copy");
    field.remove();
  }

  let toastTimer;
  function showToast(message) {
    const toast = el("toast");
    toast.textContent = message;
    toast.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toast.hidden = true; }, 2600);
  }

  function initials(name) {
    return (name || "?").split(/[\s-]+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
  }

  function setDeviceMark(mark, name) {
    mark.replaceChildren(icon("device"));
    const node = document.createElement("span");
    node.textContent = initials(name);
    mark.append(node);
  }

  function statusClass(status) {
    return ["reachable", "connected", "waiting", "reconnecting", "pairing-required", "offline"].includes(status) ? status : "offline";
  }

  function statusLabel(status) {
    return { reachable: "Available", connected: "Connected", waiting: "Waiting", reconnecting: "Reconnecting", "pairing-required": "Pairing required", offline: "Offline" }[status] || status || "Offline";
  }

  function itemKind(item) {
    if (item.type === "text") return textKind(item.content);
    return fileKind(item.filename || item.title);
  }

  function itemIconName(item) {
    if (item.type === "text") {
      const kind = textKind(item.content);
      return kind === "CODE" || kind === "CMD" || kind === "JSON" ? "code" : "text";
    }
    const kind = fileKind(item.filename || item.title);
    if (kind === "IMG") return "image";
    if (kind === "ARCH") return "archive";
    if (kind === "CODE") return "code";
    return "file";
  }

  function textKind(content) {
    const value = (content || "").trim();
    if (/^(\$\s*)?(docker|kubectl|git|npm|pnpm|yarn|curl|ssh|cd|go)\b/i.test(value)) return "CMD";
    if (/^[{[]/.test(value)) return "JSON";
    if (/^(package |import |func |const |let |var |class )/m.test(value)) return "CODE";
    return "TXT";
  }

  function fileKind(filename) {
    const ext = (filename || "").split(".").pop().toUpperCase();
    if (!ext || ext === filename) return "FILE";
    if (["PNG", "JPG", "JPEG", "GIF", "WEBP", "SVG"].includes(ext)) return "IMG";
    if (["ZIP", "RAR", "7Z", "TAR", "GZ"].includes(ext)) return "ARCH";
    if (["GO", "JS", "TS", "JSON", "YAML", "YML", "MD", "HTML", "CSS", "PY", "SH"].includes(ext)) return "CODE";
    return ext.slice(0, 4);
  }

  function textPreview(value) {
    return (value || "").replace(/\s+/g, " ").trim() || "Empty text artifact";
  }

  function fileSummary(item) {
    return `${formatBytes(item.size)} · ${fileTypeLabel(item)}`;
  }

  function fileTypeLabel(item) {
    return item.mime_type || `${fileKind(item.filename || item.title)} file`;
  }

  function folderName(folderID) {
    if (!folderID) return "Unfiled";
    const folder = state.folders.find((candidate) => candidate.id === folderID);
    return folder ? folder.name : "Folder";
  }

  function ageClass(item) {
    const days = daysSince(item.updated_at || item.created_at);
    if (days >= 60) return "very-stale";
    if (days >= 30) return "stale";
    if (days >= 14) return "aging";
    return "fresh";
  }

  function ageLabel(item) {
    const days = daysSince(item.updated_at || item.created_at);
    const age = ageClass(item);
    if (age === "very-stale") return `Very stale · ${days}d`;
    if (age === "stale") return `Stale · ${days}d`;
    if (age === "aging") return `Aging · ${days}d`;
    return days ? `Fresh · ${days}d` : "Fresh today";
  }

  function activityLabel(item) {
    return `Active ${formatRelative(item.updated_at || item.created_at)}`;
  }

  function daysSince(value) {
    const time = new Date(value).getTime();
    if (!Number.isFinite(time)) return 0;
    return Math.max(0, Math.floor((Date.now() - time) / 86400000));
  }

  function dateValue(value) {
    const time = new Date(value).getTime();
    return Number.isFinite(time) ? time : 0;
  }

  function formatRelative(value) {
    const time = dateValue(value);
    if (!time) return "recently";
    const minutes = Math.floor(Math.max(0, Date.now() - time) / 60000);
    if (minutes < 1) return "just now";
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 7) return `${days}d ago`;
    return new Date(value).toLocaleDateString([], { month: "short", day: "numeric" });
  }

  function formatDate(value) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "Unknown";
    return date.toLocaleDateString([], { year: "numeric", month: "short", day: "numeric" });
  }

  function formatBytes(size) {
    if (!size) return "0 B";
    const units = ["B", "KB", "MB", "GB"];
    let value = size;
    let index = 0;
    while (value >= 1024 && index < units.length - 1) {
      value /= 1024;
      index += 1;
    }
    return `${value.toFixed(value >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
  }

  function resultCount(count) {
    if (!state.items.length) return "No artifacts yet";
    if (count === state.items.length) return `${count} ${count === 1 ? "artifact" : "artifacts"}`;
    return `${count} of ${state.items.length} artifacts`;
  }

  function textCompare(a = "", b = "") {
    return String(a).localeCompare(String(b), undefined, { sensitivity: "base" });
  }

  document.addEventListener("DOMContentLoaded", () => {
    hydrateIcons();
    registerEvents();
    route();
  });
})();
