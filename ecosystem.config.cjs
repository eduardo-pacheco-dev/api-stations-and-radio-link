const path = require("node:path");

const deployDir =
  process.env.DEPLOY_DIR || "/root/app/api-stations-and-radio-link";
const appName = process.env.PM2_APP_NAME || "api-stations-and-radio-link";

module.exports = {
  apps: [
    {
      name: appName,
      script: path.join(deployDir, "api"),
      cwd: deployDir,
      interpreter: "none",
      exec_mode: "fork",
      instances: 1,
      autorestart: true,
      max_restarts: 10,
      restart_delay: 2000,
      time: true,
      merge_logs: true,
      out_file: path.join(deployDir, "logs", "out.log"),
      error_file: path.join(deployDir, "logs", "error.log"),
      env: {
        APP_ENV: "production",
      },
    },
  ],
};
