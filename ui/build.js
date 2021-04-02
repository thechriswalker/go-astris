const shouldWatch = process.argv.includes("--watch");
const b = () =>
  require("esbuild").build({
    entryPoints: ["./authority/index.tsx"],
    define: { "process.env.NODE_ENV": `"production"` },
    outdir: "build",
    outbase: "./",
    bundle: true,
    minify: true,
    sourcemap: true,
    sourcesContent: true,
    format: "esm",
    platform: "browser",
    watch: shouldWatch
      ? {
          onRebuild(error, result) {
            if (error) {
              console.error("watch build failed:", error);
            } else {
              console.error("watch build succeeded:", result);
            }
          },
        }
      : false,
  });

if (shouldWatch) {
  b().then(
    (result) => {
      process.on("exit", () => {
        result.stop();
      });
    },
    (err) => {
      console.error("initial build failed:", err);
      setTimeout(b, 2000);
    }
  );
} else {
  b();
}
