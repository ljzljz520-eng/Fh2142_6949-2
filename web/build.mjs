import { cp, mkdir, rm } from "node:fs/promises";

const output = new URL("./dist/", import.meta.url);

await rm(output, { recursive: true, force: true });
await mkdir(new URL("./dist/src/", import.meta.url), { recursive: true });
await Promise.all([
  cp(new URL("./index.html", import.meta.url), new URL("./dist/index.html", import.meta.url)),
  cp(new URL("./styles.css", import.meta.url), new URL("./dist/styles.css", import.meta.url)),
  cp(new URL("./src/app.js", import.meta.url), new URL("./dist/src/app.js", import.meta.url)),
]);
