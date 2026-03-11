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

  if (Auth.isEnabled()) {
    const account = Auth.getAccount();
    const menu = document.getElementById("user-menu");
    if (account && menu) {
      const name = document.createElement("span");
      name.className = "user-name";
      name.textContent = account.name ?? account.username;

      const logout = document.createElement("button");
      logout.className = "user-logout";
      logout.textContent = "Logout";
      logout.addEventListener("click", () => Auth.logout());

      menu.append(name, logout);
    }
  }
})();
