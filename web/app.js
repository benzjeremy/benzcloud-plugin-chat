const i18n = {
  de: {
    badge_realtime: "Echtzeit Mesh-Chat",
    label_user: "Name:",
    title_channels: "KANÄLE",
    title_online: "ONLINE",
    status_connected: "Verbunden",
    status_connecting: "Verbinde...",
    status_disconnected: "Getrennt",
    placeholder_input: "Nachricht an #{channel} schreiben...",
    btn_send: "Senden",
    modal_new_channel_title: "Neuen Kanal erstellen",
    label_channel_name: "Kanalname:",
    label_channel_topic: "Thema / Beschreibung:",
    btn_cancel: "Abbrechen",
    btn_create: "Erstellen",
    no_messages: "Keine Nachrichten bisher. Sei der Erste!",
  },
  en: {
    badge_realtime: "Realtime Mesh Chat",
    label_user: "Name:",
    title_channels: "CHANNELS",
    title_online: "ONLINE",
    status_connected: "Connected",
    status_connecting: "Connecting...",
    status_disconnected: "Disconnected",
    placeholder_input: "Message #{channel}...",
    btn_send: "Send",
    modal_new_channel_title: "Create New Channel",
    label_channel_name: "Channel Name:",
    label_channel_topic: "Topic / Description:",
    btn_cancel: "Cancel",
    btn_create: "Create",
    no_messages: "No messages yet. Say hello!",
  }
};

let currentLang = localStorage.getItem("benzcloud_lang") || "de";
let currentUser = localStorage.getItem("benzcloud_chat_user") || "admin";
let currentChannel = "general";
let channels = [];
let onlineUsers = [];
let ws = null;
let reconnectTimer = null;

const AVATAR_COLORS = [
  "#38bdf8", "#818cf8", "#c084fc", "#f472b6", "#fb7185",
  "#34d399", "#4ade80", "#fbbf24", "#f87171"
];

function getAvatarColor(name) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

function setLang(lang) {
  currentLang = lang;
  localStorage.setItem("benzcloud_lang", lang);
  document.documentElement.lang = lang;

  document.querySelectorAll("[data-i18n]").forEach(el => {
    const key = el.getAttribute("data-i18n");
    if (i18n[lang] && i18n[lang][key]) {
      el.textContent = i18n[lang][key];
    }
  });

  document.getElementById("btnDe").classList.toggle("active", lang === "de");
  document.getElementById("btnEn").classList.toggle("active", lang === "en");

  updateInputPlaceholder();
}

function updateInputPlaceholder() {
  const input = document.getElementById("chatInput");
  input.placeholder = i18n[currentLang].placeholder_input.replace("{channel}", currentChannel);
}

// WebSocket Connection
function connectWS() {
  if (ws) {
    ws.close();
  }

  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${proto}//${window.location.host}/ws?user=${encodeURIComponent(currentUser)}`;

  const statusEl = document.getElementById("wsStatus");
  const labelEl = statusEl.querySelector(".status-label");
  statusEl.className = "connection-status";
  labelEl.textContent = i18n[currentLang].status_connecting;

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    statusEl.className = "connection-status";
    labelEl.textContent = i18n[currentLang].status_connected;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  };

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.type === "presence") {
        updatePresence(data.users);
      } else if (data.channel) {
        if (data.channel.toLowerCase() === currentChannel.toLowerCase()) {
          appendMessage(data);
        }
      }
    } catch (e) {
      console.error("Error parsing WS message", e);
    }
  };

  ws.onclose = () => {
    statusEl.className = "connection-status disconnected";
    labelEl.textContent = i18n[currentLang].status_disconnected;
    if (!reconnectTimer) {
      reconnectTimer = setTimeout(connectWS, 3000);
    }
  };

  ws.onerror = () => {
    ws.close();
  };
}

function updatePresence(users) {
  onlineUsers = users || [];
  document.getElementById("onlineCount").textContent = onlineUsers.length;

  const container = document.getElementById("onlineUsersList");
  container.innerHTML = "";

  onlineUsers.forEach(u => {
    const item = document.createElement("div");
    item.className = "user-item";
    item.innerHTML = `
      <span class="status-indicator"></span>
      <span class="user-handle">${escapeHtml(u)}</span>
    `;
    container.appendChild(item);
  });
}

// Channels & Messages
async function loadChannels() {
  try {
    const res = await fetch("/api/chat/channels");
    if (res.ok) {
      channels = await res.json();
      renderChannels();
    }
  } catch (err) {
    console.error("Failed to load channels", err);
  }
}

function renderChannels() {
  const list = document.getElementById("channelList");
  list.innerHTML = "";

  channels.forEach(ch => {
    const btn = document.createElement("button");
    btn.className = `channel-item ${ch.name.toLowerCase() === currentChannel.toLowerCase() ? "active" : ""}`;
    btn.innerHTML = `<span class="channel-prefix">#</span> ${escapeHtml(ch.name)}`;
    btn.addEventListener("click", () => switchChannel(ch.name));
    list.appendChild(btn);
  });

  const activeObj = channels.find(c => c.name.toLowerCase() === currentChannel.toLowerCase());
  if (activeObj) {
    document.getElementById("channelHeaderTitle").textContent = `#${activeObj.name}`;
    document.getElementById("channelHeaderTopic").textContent = activeObj.topic || "";
  }
  updateInputPlaceholder();
}

async function switchChannel(channelName) {
  currentChannel = channelName.toLowerCase();
  renderChannels();
  await loadMessages();
}

async function loadMessages() {
  const stream = document.getElementById("messageStream");
  stream.innerHTML = "";

  try {
    const res = await fetch(`/api/chat/messages?channel=${encodeURIComponent(currentChannel)}&limit=100`);
    if (res.ok) {
      const messages = await res.json();
      if (!messages || messages.length === 0) {
        stream.innerHTML = `<div class="empty-state" style="color: var(--text-subtle); text-align: center; margin-top: 2rem;">${i18n[currentLang].no_messages}</div>`;
      } else {
        messages.forEach(m => appendMessage(m, false));
        scrollToBottom();
      }
    }
  } catch (err) {
    console.error("Failed to load messages", err);
  }
}

function appendMessage(msg, scroll = true) {
  const stream = document.getElementById("messageStream");
  const emptyState = stream.querySelector(".empty-state");
  if (emptyState) emptyState.remove();

  const row = document.createElement("div");
  row.className = "message-row";

  const sender = msg.sender || "Anonymous";
  const initials = sender.substring(0, 2).toUpperCase();
  const avatarColor = getAvatarColor(sender);

  const timeStr = new Date(msg.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });

  row.innerHTML = `
    <div class="avatar" style="background-color: ${avatarColor}">${initials}</div>
    <div class="message-content">
      <div class="message-meta">
        <span class="sender-name">${escapeHtml(sender)}</span>
        <span class="msg-timestamp">${timeStr}</span>
      </div>
      <div class="msg-text">${escapeHtml(msg.content)}</div>
    </div>
  `;

  stream.appendChild(row);
  if (scroll) {
    scrollToBottom();
  }
}

function scrollToBottom() {
  const stream = document.getElementById("messageStream");
  stream.scrollTop = stream.scrollHeight;
}

function sendMessage(e) {
  e.preventDefault();
  const input = document.getElementById("chatInput");
  const text = input.value.trim();
  if (!text) return;

  const payload = {
    channel: currentChannel,
    sender: currentUser,
    content: text,
    type: "message",
    timestamp: new Date().toISOString()
  };

  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(payload));
    input.value = "";
  } else {
    // Fallback via HTTP API if WS is disconnected
    fetch("/api/chat/messages", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    }).then(() => {
      input.value = "";
      loadMessages();
    });
  }
}

// Modal handling
function openModal() {
  document.getElementById("channelModal").style.display = "flex";
  document.getElementById("newChannelName").value = "";
  document.getElementById("newChannelTopic").value = "";
  document.getElementById("newChannelName").focus();
}

function closeModal() {
  document.getElementById("channelModal").style.display = "none";
}

async function handleCreateChannel(e) {
  e.preventDefault();
  const name = document.getElementById("newChannelName").value.trim().toLowerCase();
  const topic = document.getElementById("newChannelTopic").value.trim();

  if (!name) return;

  try {
    const res = await fetch("/api/chat/channels", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, topic, created_by: currentUser })
    });
    if (res.ok) {
      closeModal();
      await loadChannels();
      switchChannel(name);
    }
  } catch (err) {
    alert("Error creating channel: " + err.message);
  }
}

function escapeHtml(str) {
  return (str || "").replace(/[&<>"']/g, function (m) {
    return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[m];
  });
}

// Init
document.addEventListener("DOMContentLoaded", async () => {
  setLang(currentLang);

  document.getElementById("btnDe").addEventListener("click", () => setLang("de"));
  document.getElementById("btnEn").addEventListener("click", () => setLang("en"));

  const userField = document.getElementById("usernameInput");
  userField.value = currentUser;
  userField.addEventListener("change", (e) => {
    currentUser = e.target.value.trim() || "admin";
    localStorage.setItem("benzcloud_chat_user", currentUser);
    connectWS();
  });

  document.getElementById("chatForm").addEventListener("submit", sendMessage);
  document.getElementById("btnNewChannel").addEventListener("click", openModal);
  document.getElementById("btnCloseModal").addEventListener("click", closeModal);
  document.getElementById("btnCancelChannel").addEventListener("click", closeModal);
  document.getElementById("newChannelForm").addEventListener("submit", handleCreateChannel);

  await loadChannels();
  await loadMessages();
  connectWS();
});
