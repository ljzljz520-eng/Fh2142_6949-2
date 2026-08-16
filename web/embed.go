package web

import "embed"

//go:embed index.html styles.css src/app.js
var Files embed.FS
