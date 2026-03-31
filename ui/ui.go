package ui

import "embed"

//go:embed index.html style.css app.js auth.js sidebar.js collections.js boost.js docs.js modals.js state.js utils.js imports.js
var Files embed.FS
