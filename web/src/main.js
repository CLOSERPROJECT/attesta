import "./styles.css";

const themeStorageKey = "attesta_theme";
const themeToggle = document.getElementById("theme-toggle");

const syncFormataDarkMode = (theme) => {
  const isDark = theme === "dark";
  const components = document.querySelectorAll("formata-form.js-formata-form");
  for (const component of components) {
    if (isDark) {
      component.setAttribute("dark-mode", "");
    } else {
      component.removeAttribute("dark-mode");
    }
  }
};

const getPreferredTheme = () => {
  try {
    const stored = localStorage.getItem(themeStorageKey);
    if (stored === "light" || stored === "dark") {
      return stored;
    }
  } catch (err) {
  }

  if (
    window.matchMedia &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  ) {
    return "dark";
  }

  return "light";
};

const applyTheme = (theme) => {
  document.documentElement.dataset.theme = theme;
  syncFormataDarkMode(theme);
  if (themeToggle) {
    themeToggle.textContent = theme === "dark" ? "Light" : "Dark";
  }
};

const setTheme = (theme) => {
  applyTheme(theme);
  try {
    localStorage.setItem(themeStorageKey, theme);
  } catch (err) {
  }
};

applyTheme(getPreferredTheme());

if (themeToggle) {
  themeToggle.addEventListener("click", () => {
    const current = document.documentElement.dataset.theme;
    setTheme(current === "dark" ? "light" : "dark");
  });
}

const root = document.querySelector("[data-process-id]");
const processId = root?.dataset?.processId;
const workflowKey = root?.dataset?.workflowKey;
const timeline = document.getElementById("timeline");
const processDownloads = document.getElementById("process-downloads");

const focusNextActionInput = () => {
  const nextInput = document.querySelector(
    ".action-card.action-available:not(.action-other) input:not([disabled]):not([type='hidden']), .action-card.action-available:not(.action-other) textarea:not([disabled]), .action-card.action-available:not(.action-other) select:not([disabled])"
  );
  if (nextInput) {
    nextInput.focus();
  }
};

const resolveAbsoluteURL = (value) => {
  try {
    return new URL(value, window.location.origin).toString();
  } catch (_err) {
    return value;
  }
};

const shareLink = async (button) => {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }
  const rawURL = button.dataset.shareUrl;
  if (!rawURL) {
    return;
  }
  const absoluteURL = resolveAbsoluteURL(rawURL);
  const shareLabel = button.dataset.shareLabel || "link";
  if (navigator.share) {
    try {
      await navigator.share({ url: absoluteURL });
      return;
    } catch (err) {
      if (err && typeof err === "object" && "name" in err && err.name === "AbortError") {
        return;
      }
    }
  }
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(absoluteURL);
      const originalText = button.textContent || `Share ${shareLabel}`;
      button.textContent = "Copied";
      window.setTimeout(() => {
        button.textContent = originalText;
      }, 1200);
      return;
    } catch (_err) {
    }
  }
  window.prompt(`Copy ${shareLabel}:`, absoluteURL);
};

const copyTextValue = async (button) => {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }
  const value = button.dataset.copyText || "";
  if (!value) {
    return;
  }
  const label = button.dataset.copyLabel || "value";
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      const originalText = button.textContent || "Copy";
      button.textContent = "Copied";
      window.setTimeout(() => {
        button.textContent = originalText;
      }, 1200);
      return;
    } catch (_err) {
    }
  }
  window.prompt(`Copy ${label}:`, value);
};

const parseJsonAttribute = (value) => {
  if (!value) {
    return undefined;
  }
  try {
    return JSON.parse(value);
  } catch (_err) {
    return undefined;
  }
};

const normalizeFormataSchema = (schema) => {
  if (!schema || typeof schema !== "object" || Array.isArray(schema)) {
    return undefined;
  }
  const normalized = { ...schema };
  if (!normalized.type) {
    normalized.type = "object";
  }
  if (normalized.type === "object" && (!normalized.properties || typeof normalized.properties !== "object")) {
    normalized.properties = {};
  }
  if (normalized.required && !Array.isArray(normalized.required)) {
    delete normalized.required;
  }
  return normalized;
};

const normalizeUiOptions = (options) => {
  if (!options || typeof options !== "object" || Array.isArray(options)) {
    return {};
  }
  return { ...options };
};

const normalizeFieldUiSchema = (fieldUi) => {
  if (!fieldUi || typeof fieldUi !== "object" || Array.isArray(fieldUi)) {
    return {};
  }
  const normalized = { ...fieldUi };
  const options = normalizeUiOptions(normalized["ui:options"]);
  const description = normalized["ui:description"];
  const help = normalized["ui:help"];
  if (typeof description === "string" && description.trim() !== "" && typeof options.description !== "string") {
    options.description = description;
  }
  if (typeof help === "string" && help.trim() !== "" && typeof options.help !== "string") {
    options.help = help;
  }
  if (Object.keys(options).length > 0) {
    normalized["ui:options"] = options;
  } else {
    delete normalized["ui:options"];
  }
  delete normalized["ui:description"];
  delete normalized["ui:help"];
  const widget = normalized["ui:widget"];
  if (widget) {
    const components = { ...(normalized["ui:components"] ?? {}) };
    if (widget === "textarea") {
      components.textWidget = components.textWidget ?? "textareaWidget";
    } else if (widget === "file") {
      components.stringField = components.stringField ?? "fileField";
    } else if (widget === "range") {
      components.numberWidget = components.numberWidget ?? "rangeWidget";
    }
    normalized["ui:components"] = components;
    delete normalized["ui:widget"];
  }
  return normalized;
};

const normalizeFormataUiSchema = (uiSchema) => {
  if (!uiSchema || typeof uiSchema !== "object" || Array.isArray(uiSchema)) {
    return {};
  }
  const source = uiSchema.properties && typeof uiSchema.properties === "object" ? uiSchema.properties : uiSchema;
  const normalized = {};
  const rootOptions = normalizeUiOptions(uiSchema["ui:options"]);
  if (Array.isArray(uiSchema["ui:order"])) {
    rootOptions.order = uiSchema["ui:order"];
  }
  if (Object.keys(rootOptions).length > 0) {
    normalized["ui:options"] = rootOptions;
  }
  for (const [key, value] of Object.entries(uiSchema)) {
    if (key === "ui:options" || key === "ui:order" || key === "properties") {
      continue;
    }
    if (key.startsWith("ui:")) {
      normalized[key] = value;
    }
  }
  for (const [key, value] of Object.entries(source)) {
    if (key !== "properties" && !key.startsWith("ui:")) {
      normalized[key] = normalizeFieldUiSchema(value);
    }
  }
  return normalized;
};

const fileToDataURL = (file) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener("load", () => resolve(reader.result), { once: true });
    reader.addEventListener("error", () => reject(new Error("failed to read file")), { once: true });
    reader.readAsDataURL(file);
  });

const serializeFormataValue = async (value) => {
  if (value instanceof File) {
    try {
      const dataURL = await fileToDataURL(value);
      if (typeof dataURL === "string" && dataURL.startsWith("data:")) {
        return dataURL;
      }
    } catch (_err) {
    }
    return "";
  }
  if (Array.isArray(value)) {
    const normalized = [];
    for (const entry of value) {
      normalized.push(await serializeFormataValue(entry));
    }
    return normalized;
  }
  if (value && typeof value === "object") {
    const normalized = {};
    for (const [key, entry] of Object.entries(value)) {
      normalized[key] = await serializeFormataValue(entry);
    }
    return normalized;
  }
  return value;
};

const formDataToObject = async (formData) => {
  const result = {};
  for (const [key, value] of formData.entries()) {
    const normalizedValue = await serializeFormataValue(value);
    if (Object.prototype.hasOwnProperty.call(result, key)) {
      if (Array.isArray(result[key])) {
        result[key].push(normalizedValue);
      } else {
        result[key] = [result[key], normalizedValue];
      }
    } else {
      result[key] = normalizedValue;
    }
  }
  return result;
};

const readFormataComponentValue = (component) => {
  if (typeof component?.getValue !== "function") {
    return {};
  }
  const value = component.getValue();
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value;
  }
  return {};
};

const extractFormataSubmitPayload = async (componentEvent, component) => {
  const detail = componentEvent instanceof CustomEvent ? componentEvent.detail : undefined;
  if (detail instanceof FormData) {
    return await formDataToObject(detail);
  }
  if (detail && typeof detail === "object" && detail.formData instanceof FormData) {
    return await formDataToObject(detail.formData);
  }
  if (detail && typeof detail === "object" && !Array.isArray(detail)) {
    return await serializeFormataValue(detail);
  }
  return await serializeFormataValue(readFormataComponentValue(component));
};

let formataReadyPromise;
const formataScriptURLs = [
  "https://cdn.jsdelivr.net/gh/CLOSERPROJECT/formata@main/dist/formata-web-component.umd.js",
  "https://closerproject.github.io/formata-arch/formata-web-component.umd.js"
];

const waitForFormataDefinition = (timeoutMs) => {
  if (customElements.get("formata-form")) {
    return Promise.resolve(true);
  }
  return Promise.race([
    customElements.whenDefined("formata-form").then(() => true),
    new Promise((resolve) => setTimeout(() => resolve(false), timeoutMs))
  ]);
};

const loadExternalScript = (url) =>
  new Promise((resolve) => {
    const existing = document.querySelector(`script[data-formata-src="${url}"]`);
    if (existing) {
      resolve(true);
      return;
    }
    const script = document.createElement("script");
    script.src = url;
    script.defer = true;
    script.dataset.formataSrc = url;
    script.addEventListener("load", () => resolve(true), { once: true });
    script.addEventListener("error", () => resolve(false), { once: true });
    document.head.appendChild(script);
  });

const ensureFormataLoaded = async () => {
  if (customElements.get("formata-form")) {
    return true;
  }
  for (const url of formataScriptURLs) {
    const loaded = await loadExternalScript(url);
    if (!loaded) {
      continue;
    }
    const ready = await waitForFormataDefinition(2500);
    if (ready) {
      return true;
    }
  }
  return false;
};

const waitForFormata = async () => {
  if (customElements.get("formata-form")) {
    return true;
  }
  if (!formataReadyPromise) {
    formataReadyPromise = ensureFormataLoaded();
  }
  return await Promise.race([
    formataReadyPromise,
    new Promise((resolve) => setTimeout(() => resolve(false), 5000))
  ]);
};

const syncFormataValue = (component, hiddenInput) => {
  hiddenInput.value = JSON.stringify(readFormataComponentValue(component));
};

const submitFormataPayload = (form, hiddenInput, payload) => {
  const state = form.dataset.formataSubmitState || "idle";
  if (state !== "idle") {
    return false;
  }
  const serialized = JSON.stringify(payload ?? {});
  hiddenInput.value = serialized;
  form.dataset.formataSubmitState = "inflight";

  const url = form.dataset.formataPost || form.getAttribute("hx-post") || form.getAttribute("action");
  const htmxApi = window.htmx;
  if (!url || !htmxApi || typeof htmxApi.ajax !== "function") {
    form.dataset.formataSubmitState = "idle";
    return false;
  }
  htmxApi.ajax("POST", url, {
    source: form,
    target: "#action-area",
    swap: "innerHTML",
    values: { value: serialized }
  });
  return true;
};

const initializeFormataForms = async (container = document) => {
  const hosts = container.querySelectorAll(".js-formata-host");
  if (hosts.length === 0) {
    return;
  }
  const formataReady = await waitForFormata();
  if (!formataReady) {
    return;
  }
  for (const host of hosts) {
    if (host.dataset.formataReady === "true") {
      continue;
    }
    host.dataset.formataReady = "true";

    const rawSchema = parseJsonAttribute(host.getAttribute("data-formata-schema"));
    const schema = normalizeFormataSchema(rawSchema);
    const rawUiSchema = parseJsonAttribute(host.getAttribute("data-formata-uischema"));
    const uiSchema = normalizeFormataUiSchema(rawUiSchema);
    if (!schema) {
      continue;
    }

    const component = document.createElement("formata-form");
    component.className = "js-formata-form";
    component.setAttribute("prevent-page-reload", "");
    if (document.documentElement.dataset.theme === "dark") {
      component.setAttribute("dark-mode", "");
    }
    if (host.dataset.formataDisabled === "true") {
      component.setAttribute("disabled", "");
    }
    component.schema = schema;
    component.uiSchema = uiSchema;
    host.appendChild(component);

    const form = component.closest("form");
    if (!form) {
      continue;
    }
    const hiddenInput = form.querySelector(".js-formata-value");
    if (!(hiddenInput instanceof HTMLInputElement)) {
      continue;
    }

    const sync = () => syncFormataValue(component, hiddenInput);
    sync();
    component.addEventListener("formata:change", sync);
    component.addEventListener("submit", async (componentEvent) => {
      componentEvent.preventDefault?.();
      componentEvent.stopPropagation?.();
      const state = form.dataset.formataSubmitState || "idle";
      if (state !== "idle" || host.dataset.formataDisabled === "true") {
        return;
      }
      let payload = {};
      try {
        payload = await extractFormataSubmitPayload(componentEvent, component);
      } catch (_err) {
        payload = await serializeFormataValue(readFormataComponentValue(component));
      }
      if (!submitFormataPayload(form, hiddenInput, payload)) {
        form.dataset.formataSubmitState = "idle";
      }
    });
    form.addEventListener("submit", async (submitEvent) => {
      submitEvent.preventDefault();
      if (submitEvent.target === component) {
        return;
      }
      const state = form.dataset.formataSubmitState || "idle";
      if (state !== "idle" || host.dataset.formataDisabled === "true") {
        return;
      }
      let payload = {};
      try {
        payload = await serializeFormataValue(readFormataComponentValue(component));
      } catch (_err) {
        payload = readFormataComponentValue(component);
      }
      if (!submitFormataPayload(form, hiddenInput, payload)) {
        form.dataset.formataSubmitState = "idle";
      }
    });

    form.addEventListener("htmx:afterRequest", () => {
      delete form.dataset.formataSubmitState;
    });
    form.addEventListener("htmx:responseError", () => {
      delete form.dataset.formataSubmitState;
    });
  }
};

if (processId && workflowKey && timeline) {
  const source = new EventSource(
    `/w/${workflowKey}/events?workflow=${workflowKey}&processId=${processId}`
  );
  source.addEventListener("process-updated", async () => {
    try {
      const response = await fetch(`/w/${workflowKey}/process/${processId}/timeline`);
      if (!response.ok) {
        return;
      }
      const html = await response.text();
      timeline.innerHTML = html;
    } catch (err) {
      // keep UI responsive even if SSE fails
    }
    if (processDownloads) {
      try {
        const response = await fetch(`/w/${workflowKey}/process/${processId}/downloads`);
        if (!response.ok) {
          return;
        }
        const html = await response.text();
        const current = document.getElementById("process-downloads");
        if (current) {
          current.outerHTML = html;
        }
      } catch (err) {
        // keep UI responsive even if SSE fails
      }
    }
  });
}

document.addEventListener("DOMContentLoaded", () => {
  void initializeFormataForms(document);
  focusNextActionInput();
});

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.target && event.target.id === "action-area") {
    void initializeFormataForms(event.target);
    focusNextActionInput();
  }
});

document.body.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof Element)) {
    return;
  }
  const copyButton = target.closest(".js-copy-text");
  if (copyButton instanceof HTMLButtonElement) {
    event.preventDefault();
    void copyTextValue(copyButton);
    return;
  }
  const shareButton = target.closest(".js-share-link");
  if (!(shareButton instanceof HTMLButtonElement)) {
    return;
  }
  event.preventDefault();
  void shareLink(shareButton);
});

const deptRoot = document.querySelector("[data-dept-role]");
if (deptRoot) {
  const role = deptRoot.dataset.deptRole;
  const deptWorkflowKey = deptRoot.dataset.workflowKey;
  const dashboard = document.getElementById("dept-dashboard");
  if (role && deptWorkflowKey && dashboard) {
    const source = new EventSource(
      `/w/${deptWorkflowKey}/events?workflow=${deptWorkflowKey}&role=${role}`
    );
    source.addEventListener("role-updated", async () => {
      try {
        const response = await fetch(`/w/${deptWorkflowKey}/backoffice/${role}/partial`);
        if (!response.ok) {
          return;
        }
        const html = await response.text();
        dashboard.innerHTML = html;
      } catch (err) {
        // keep UI responsive even if SSE fails
      }
    });
  }
}
