# BenzCloud Chat Plugin (`benzcloud-plugin-chat`)

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![Version](https://img.shields.io/badge/Version-v1.0-green.svg)](https://github.com/benzjeremy/benzcloud-plugin-chat)

The **BenzCloud Chat Plugin** provides a real-time, decentralized team chat service for the BenzCloud mesh ecosystem. Designed as a zero-dependency, self-hosted alternative to Slack and Discord, it operates purely over local encrypted peer-to-peer networks.

---

## 🌟 Key Features

- **Real-Time WebSocket Architecture:** Low-latency bi-directional messaging with instant delivery.
- **Dynamic Channels & Topics:** Pre-configured `#general`, `#dev`, `#random` rooms with on-the-fly channel creation.
- **Live User Presence:** Instant peer online/offline tracking across the local mesh.
- **Embedded Web UI:** Cybernetic Dark interface inspired by modern team chat tools with a dual-language toggle (DE / EN), avatar badge generation, and responsive mobile-friendly layout.
- **BenzCloud Service Integration:** Auto-discovers with `benzcloud-server` via `/health` API for automatic DNS mapping (`chat.<domain>`) and reverse-proxy routing.
- **Decentralized Persistence:** Zero cloud telemetry. All message logs and channels persist locally in structured JSON records.

---

## 🚀 Quick Start

### Running the Plugin
```bash
./benzcloud-plugin-chat -port=8093 -data=./data -domain=benzcloud.local
```

### Command-line Options
- `-port <number>`: Web UI and WebSocket server port (default: `8093`).
- `-data <path>`: Local storage directory for chat history (default: `./data`).
- `-domain <name>`: Base mesh domain (default: `benzcloud.local`).

---

## 🛡️ License

This project is licensed under the [GNU General Public License v3.0 (GPL-3.0)](LICENSE).
