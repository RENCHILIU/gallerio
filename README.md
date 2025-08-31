# SlideHub

A lightweight **LAN photo slideshow** service built with **Go (net/http)** and **MySQL** using **database/sql (raw SQL)**. 
Browse photos from any device on your local network, upload new ones from phone/desktop, and play a fullscreen slideshow with *no upscaling* and a *blurred background*—matching your previous HTML prototype.

## ✨ MVP Scope
- Photo grid + infinite scroll (keyset pagination)
- Multi-file uploads (multipart/form-data) to disk
- Simple MySQL schema (`photos` table) with raw SQL
- Slideshow player: fullscreen, interval control, prev/next, pause, **only scale down**, blurred cover background
- Local network only (no auth, no public deploy) for now

## 🧰 Tech
- Go 1.22+ — `net/http`, `database/sql`
- MySQL 8 — existing instance on your Linux server (`localhost:3306`)
- HTML templates + vanilla JS
- OpenAPI (Swagger UI under `/docs`) for API contract

## ⚙️ Requirements
- Go 1.22+
- MySQL 8 (database: `slideshow`, user with minimal privileges)


## 🔐 Out-of-scope (Backlog)
- Auth (login), observability, CI/CD, public deploy, EXIF/derivatives
