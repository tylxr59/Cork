<p align="center">
  <img src="assets/logo.png" alt="Cork logo" width="112" height="112">
</p>

<h1 align="center">Cork</h1>

<p align="center">
  A self-hosted pinboard for saving images and ideas into boards, made for homelabs and small groups.
</p>

<p align="center">
  <a href="https://github.com/tylxr59/Cork/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/tylxr59/Cork?display_name=tag&sort=semver"></a>
  <a href="LICENSE"><img alt="MIT license" src="https://img.shields.io/badge/license-MIT-97ca00"></a>
</p>

<p align="center">
  <a href="#-features">Features</a> ·
  <a href="#-quick-start">Quick start</a> ·
  <a href="#-unraid">Unraid</a> ·
  <a href="#-usage">Usage</a> ·
  <a href="#%EF%B8%8F-configuration">Configuration</a> ·
  <a href="#-troubleshooting">Troubleshooting</a>
</p>

---

Cork is a Pinterest-style board for your own server. Save images from your computer or from any website, sort them into boards, drag them into the order you want, and keep the link back to where each one came from. Everything lives in one SQLite file and one image folder, so backups are easy, and none of it leaves your network.

> [!NOTE]
> Cork is for personal and small-group use. It has no social feed, follows or recommendations, and it never will.

## 🎯 Features

| Feature                       | What it does                                                                                     |
| ----------------------------- | ------------------------------------------------------------------------------------------------ |
| 📌 **Boards and pins**        | Each pin has an image, title, description, source link and the date it was added.                |
| ↕️ **Drag to reorder**        | Rearrange pins inside a board, and rearrange the boards themselves.                              |
| 🌐 **Save from websites**     | Paste a link and Cork lists the page's images, title and description so you can pick one.        |
| 🔖 **Bookmarklet**            | "Pin to Cork" in your bookmarks bar pins from whatever page you're on.                           |
| 🏷️ **Prices**                 | Pins from shop pages save the product's price as it was the day you pinned it.                   |
| 📥 **Upload, paste or drop**  | Paste an image from the clipboard, drop files, or upload several images at once.                 |
| 🗃️ **Mass import**            | Drop a pile of images or a whole folder on a board and each one becomes a pin.                   |
| 👥 **Multiple users**         | Each user has their own private boards. Admins manage accounts and sign-ups.                     |
| 🔗 **Public boards**          | Share a board with a link. Visitors can view it without an account.                              |
| 🔎 **Search**                 | Searches pin titles, descriptions, links and board names.                                        |
| 🐳 **One small container**    | A single ~25 MB Go binary with the web app built in. No database server needed.                 |
| 🌙 **Light and dark**         | Follows your system theme, or pick one in the account menu. Works on phones, tablets and desktops. |

## 📋 Requirements

- Docker (or any OCI runtime) on `linux/amd64` or `linux/arm64`. You can also run the release binary on its own.
- About 30 MB of RAM at idle, plus disk space for your images.

## 🚀 Quick start

```sh
curl -O https://raw.githubusercontent.com/tylxr59/Cork/main/docker-compose.yml
docker compose up -d
```

Then open `http://<your-server>:8080`. The first account you create becomes the admin.

Or with plain Docker:

```sh
docker run -d --name cork -p 8080:8080 -v cork-data:/data --restart unless-stopped ghcr.io/tylxr59/cork:latest
```

## 🟧 Unraid

### Option A: use the template (fastest)

1. Open the Unraid terminal (the `>_` icon in the top bar) and run:

   ```sh
   wget -O /boot/config/plugins/dockerMan/templates-user/my-Cork.xml \
     https://raw.githubusercontent.com/tylxr59/Cork/main/unraid/cork.xml
   ```

2. Go to **Docker → Add Container**, choose **Cork** from the **Template** dropdown, check the values, and click **Apply**.

### Option B: fill in Add Container by hand

Go to **Docker → Add Container** and switch on **Advanced View** (top right), so you can see Extra Parameters. Then fill in:

| Field                  | Value                                                                   |
| ---------------------- | ----------------------------------------------------------------------- |
| **Name**               | `Cork`                                                                  |
| **Overview**           | `Self-hosted pinboard for images, links and ideas.` (optional)          |
| **Repository**         | `ghcr.io/tylxr59/cork:latest`                                           |
| **Registry URL**       | `https://github.com/tylxr59/Cork/pkgs/container/cork` (optional)        |
| **Icon URL**           | `https://raw.githubusercontent.com/tylxr59/Cork/main/assets/logo.png`   |
| **WebUI**              | `http://[IP]:[PORT:8080]/`                                              |
| **Extra Parameters**   | `--user 99:100` (**required**, see below)                               |
| **Post Arguments**     | leave empty                                                             |
| **Network Type**       | `Bridge`                                                                |
| **Console shell command** | leave the default. The image has no shell, so the console won't open, but Cork doesn't need one. |
| **Privileged**         | `Off`                                                                   |

Then use **Add another Path, Port, Variable, Label or Device** to add each of these:

| Config Type  | Name                   | Container value        | Host value / default        | Notes                                                    |
| ------------ | ---------------------- | ---------------------- | --------------------------- | -------------------------------------------------------- |
| **Port**     | WebUI port             | `8080` (TCP)           | `8080`                      | Change the host side if 8080 is already taken.           |
| **Path**     | Data                   | `/data`                | `/mnt/user/appdata/cork`    | Access mode **Read/Write**. Holds the database and images. |
| **Variable** | Behind a reverse proxy | `CORK_TRUST_PROXY`     | `false`                     | Set to `true` if you use SWAG, Nginx Proxy Manager, Traefik and the like. |
| **Variable** | Admin username         | `CORK_ADMIN_USERNAME`  | *(empty)*                   | Optional. Creates the admin on first start.              |
| **Variable** | Admin password         | `CORK_ADMIN_PASSWORD`  | *(empty)*                   | Optional. Turn on **Mask** so it's hidden.               |

All the other [configuration variables](#%EF%B8%8F-configuration) work the same way: add them as Variables.

Click **Apply**, then open the WebUI. If you didn't set the admin variables, the first account you create becomes the admin.

> [!IMPORTANT]
> **Why `--user 99:100`?** Cork doesn't run as root. Unraid's `appdata` folders belong to `nobody:users` (UID 99, GID 100), so this setting lets Cork write there, and its files get the same owner as the rest of your appdata. Without it, the container stops on start with `data directory /data is not writable`.

**Backups:** the whole install is `/mnt/user/appdata/cork`. The Appdata Backup plugin picks it up automatically.

**Updates:** Cork shows up in the Docker tab's update check like any other container. Click **apply update**, and your data stays put.

## 🧭 Usage

- **Create a board** from the Boards page, then use **Create** in the header to add pins.
- **Add images** by choosing files, dragging them onto the page, or pasting from the clipboard (<kbd>Ctrl</kbd>/<kbd>⌘</kbd>+<kbd>V</kbd>). If you add several images, each one becomes its own pin.
- **Save from a website** by pasting the page's URL into "Save from a website" (or anywhere on the Create page). Cork fills in the title, description and link, and shows the page's images for you to choose from.
- **Bookmarklet:** open **Settings** and drag **Pin to Cork** to your bookmarks bar. On any site, click it to open Cork in a small window with that page ready to pin. If the page has several images, tick the ones you want and each becomes its own pin. It reads images in your browser, so it works on sites that block Cork's server, like Amazon and Reddit. If you added the bookmarklet before this picker existed, drag it to your bookmarks bar again.
- **Prices:** when you save a product, Cork records its price and currency from the page's structured data (schema.org `Product` offers, `product:price` meta tags or microdata). The bookmarklet reads the price in your browser, so it also works on shops that block automated requests or render prices with JavaScript (Amazon included). When a page has no structured price it looks for known shop layouts, then the most prominent price on the page, skipping crossed-out "was" prices; select a price on the page before clicking it to choose one yourself. You can edit or add a price on any pin, typed however you like ("$20.23", "19,99 €").
- **Currencies:** Cork works out the currency from the price text. For symbols that several currencies share, like `$`, it goes by the site's country (amazon.ca → CAD), then the page's language, then your browser's locale; a price with no currency at all gets that same default. If it can't tell (a bare `kr`), the symbol is stored as-is.
- **Reorder** by dragging pins on a board, or boards on the Boards page. On touch screens, press and hold before dragging.
- **Share** a board with **Make public**, then **Copy link**. Public boards and their pins are read-only for everyone except you.
- **Edit a pin** to change its text, link or image, or to move it to another board.
- **Admin** (account menu): turn open sign-ups on or off, add users, reset passwords and remove accounts.

## ⚙️ Configuration

All settings are environment variables and all of them are optional.

| Variable                   | Default        | Description                                                                                                   |
| -------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------- |
| `CORK_ADDR`                | `:8080`        | Address to listen on.                                                                                         |
| `CORK_DATA_DIR`            | `/data`        | Where the database (`cork.db`) and images (`media/`) are stored. Defaults to `./data` outside Docker.         |
| `CORK_MAX_UPLOAD_MB`       | `25`           | Largest image Cork will accept, for uploads and downloads.                                                    |
| `CORK_TRUST_PROXY`         | `false`        | Set to `true` behind a reverse proxy so Cork uses `X-Forwarded-For` / `X-Forwarded-Proto` for rate limiting and secure cookies. |
| `CORK_ALLOW_PRIVATE_FETCH` | `false`        | Let "save from a website" fetch from private and LAN addresses. Leave this off if you don't trust every user. |
| `CORK_ADMIN_USERNAME`      | (none)         | With `CORK_ADMIN_PASSWORD`, creates this admin account on first start, so you can skip the setup page.       |
| `CORK_ADMIN_PASSWORD`      | (none)         | Password for that account (8 characters minimum).                                                            |

### Reverse proxy

Cork works behind any reverse proxy. Set `CORK_TRUST_PROXY=true`. Session cookies are marked `Secure` automatically when the proxy reports HTTPS. For example, with Caddy:

```caddy
cork.example.com {
  reverse_proxy cork:8080
}
```

### Backups

Stop the container (or use `sqlite3 cork.db ".backup ..."`), then copy the whole data directory. It contains `cork.db` and the `media/` folder.

## 🩺 Troubleshooting

| Symptom                                                       | Fix                                                                                                                                       |
| ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `data directory /data is not writable` on start              | The container runs as UID 65532. Use a named volume, as in the compose file, or run `sudo chown -R 65532:65532 ./data` on your bind mount. On Unraid, add `--user 99:100` to Extra Parameters (see [Unraid](#-unraid)). |
| "registration is closed"                                     | Only the first account can sign up on its own. An admin can turn on open sign-ups or add you in **Admin**.                                |
| "refusing to fetch a private or local network address"       | Cork blocks LAN addresses by default. Set `CORK_ALLOW_PRIVATE_FETCH=true` to allow them.                                                  |
| A website returns no images or an error                      | Some sites block automated requests or build their pages with JavaScript. Use the bookmarklet, or save the image and upload it.   |
| The bookmarklet does nothing                                 | Some sites block popups or bookmarklets. Allow popups for that site; if it's still blocked, Cork opens in the same tab instead.           |
| Signed out on every request behind a proxy                   | If you reach Cork over plain HTTP but the proxy sends `X-Forwarded-Proto: https`, the browser drops the secure cookie. Use HTTPS throughout. |
| Forgot the only admin password                               | Stop Cork, delete `cork.db` to start over, or reset the password with `sqlite3` and a bcrypt hash.                                       |

## 👋 Development

You need Go 1.27+ and Node 26+.

```sh
# frontend dev server with hot reload (proxies /api and /media to :8080)
cd web && npm install && npm run dev

# in another terminal: the Go server
go run ./cmd/cork

# tests and checks
go test ./...
(cd web && npm run check)

# production build: one binary with the frontend embedded
(cd web && npm run build) && go build -o cork ./cmd/cork

# or the container
docker build -t cork .
```

Project layout:

| Path                | What's there                                                               |
| ------------------- | -------------------------------------------------------------------------- |
| `cmd/cork`          | Entry point, configuration and the `healthcheck` subcommand.               |
| `internal/server`   | HTTP API, auth and sessions, SQLite schema and migrations, media serving.  |
| `internal/imaging`  | Image validation, EXIF orientation, thumbnails and placeholder colors.     |
| `internal/scrape`   | Fetching pages and images (with private-network blocking), and reading their metadata. |
| `web`               | Svelte 5 + TypeScript single-page app, embedded into the binary at build time. |

Bug reports and contributions are welcome through [GitHub Issues](https://github.com/tylxr59/Cork/issues).

## 📦 Releases

Bump `version` in `web/package.json`, then push a matching `v<version>` tag. The [release workflow](.github/workflows/release.yml) runs the tests, then:

- publishes a multi-arch image to `ghcr.io/tylxr59/cork`, tagged with the version and `latest`
- creates a [GitHub Release](https://github.com/tylxr59/Cork/releases) with standalone binaries and generated release notes

> [!NOTE]
> After the first release, open the package's settings on GitHub (**Packages → cork → Package settings**) and set its visibility to **Public**. GitHub makes new container packages private, so Docker and Unraid can't pull them until you do.

## 🤝 Acknowledgements

Cork is built on [Svelte](https://svelte.dev), [Vite](https://vite.dev), [SortableJS](https://sortablejs.github.io/Sortable/), [Lucide](https://lucide.dev) icons, and [modernc.org/sqlite](https://modernc.org/sqlite), a pure-Go SQLite.

## 📃 License

Cork is available under the [MIT License](LICENSE).
