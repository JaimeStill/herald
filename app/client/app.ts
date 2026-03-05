import { Router } from "@core/router";
import "@ui/elements";
import "@ui/modules";
import "@ui/views";

import { routes } from "./routes";

import "@design/index.css";

const router = new Router("app-content", routes);
router.start();
