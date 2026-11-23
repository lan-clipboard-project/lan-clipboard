# LAN Clipboard

**English** · [简体中文](docs/README.zh-CN.md)

Run LAN Clipboard on a computer or NAS to share text, images, and files in real time. Open the same address in a browser on any phone, tablet, or computer to view the shared clipboard history.

## Docker quick start

Run these steps on a Linux, macOS, or NAS host with Docker, Docker Compose, and `curl`. On a NAS, connect over SSH.

1. Create and enter a new deployment directory, such as `$HOME/docker/lan-clipboard`.

   ```sh
   mkdir -p "$HOME/docker/lan-clipboard" && cd "$HOME/docker/lan-clipboard"
   ```

2. Download and run the setup script.

   ```sh
   curl -fsSL -o install.sh https://github.com/lan-clipboard-project/lan-clipboard/releases/latest/download/install.sh &&
     sh ./install.sh
   ```

   The installer follows your terminal language and falls back to English. You can choose a language explicitly with `sh ./install.sh --lang en` or `sh ./install.sh --lang zh-CN`.

3. Use the arrow keys to select an option and press Enter to confirm. By default, LAN Clipboard binds to the host's local network IPv4 address. Choose **Customize settings** to set the port, time zone, retention period, file size limit, or Office previews. Choose **Yes, start now** to pull the images and start LAN Clipboard, then open the displayed URL in your browser.

Use LAN Clipboard on a trusted network, such as your home LAN. Devices that can reach the service can read, add, and delete shared items.

For manual setup, custom NAS storage, HTTPS, updates, and backups, see the [deployment guide (Simplified Chinese)](docs/deployment.md).

## Screenshots

| Desktop | Mobile |
| --- | --- |
| <img src="docs/screenshots/desktop-light.png" alt="Clipboard history on desktop" width="640"> | <img src="docs/screenshots/mobile-dark.png" alt="Mobile interface in dark mode" width="208"> |

## Features

- Paste text and images. Drag and drop files or select them to upload.
- Search clipboard history and pin frequently used items. Unpinned items are kept for 30 days by default.
- Preview images, PDFs, Markdown, and Excel files in your browser. The interface supports English, Simplified Chinese, and dark mode.

## Project links

[Changelog](CHANGELOG.md) · [Contributing](CONTRIBUTING.md) · [Security policy](SECURITY.md) · [MIT License](LICENSE) · [Third-party notices](THIRD_PARTY_NOTICES.md)
