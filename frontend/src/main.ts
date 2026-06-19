import { createApp } from "vue";
import App from "./App.vue";
import { backend } from "./backend";
import "./style.css";

const app = createApp(App);
app.config.errorHandler = (err, _instance, info) => {
  const detail = err instanceof Error ? `${err.message} | ${err.stack ?? ""}` : String(err);
  void backend.Debug("vue.error", `${info}: ${detail}`);
};
app.mount("#app");
