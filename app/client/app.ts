import { Auth } from "@core";
import { Router } from "@core/router";
import "@ui/elements";
import "@ui/modules";
import "@ui/views";

import { routes } from "./routes";

import "@design/index.css";

(async () => {
  await Auth.init();

  if (Auth.isEnabled() && !Auth.isAuthenticated()) {
    await Auth.login();
    return;
  }

  const router = new Router("app-content", routes);
  router.start();
})();
