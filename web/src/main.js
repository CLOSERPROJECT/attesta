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

const formataShadowOverrides = `
  [data-slot="field-legend"] {
    font-family: "Space Grotesk", system-ui, sans-serif;
  }
  [data-slot="field-description"] {
    font-family: "Space Grotesk", system-ui, sans-serif;
    color: var(--formata-muted) !important;
    font-size: 13px;
  }
  [data-slot="input"]::file-selector-button,
  [data-slot="button"] {
    background: var(--panel) !important;
    color: var(--formata-accent) !important;
    border: 1px solid var(--formata-accent) !important;
    cursor: pointer;
    border-radius: 4px;
  }
  [data-slot="button"][type="submit"] {
    background: var(--formata-accent) !important;
    color: var(--panel) !important;
    border-color: var(--formata-accent) !important;
    font-family: "Space Grotesk", system-ui, sans-serif;
    font-weight: 600;
    border-radius: 4px;
    cursor: pointer;
    justify-content: flex-end;
    display: flex;
    margin-left: auto;
    width: fit-content;
  }
  [data-slot="slider-range"] {
    background: var(--formata-accent) !important;
  }
  [data-slot="slider-track"] {
    background: var(--panel) !important;
  }
  [data-slot="input"],
  [data-slot="select-trigger"],
  [data-slot="select-content"],
  [data-slot="checkbox"],
  [data-slot="radio-group-item"],
  input,
  select,
  textarea {
    background: var(--panel) !important;
    border-color: var(--border) !important;
    color: var(--ink) !important;
	  font-family: "Space Grotesk", system-ui, sans-serif !important;
  }
  [data-slot="field-legend"]:has(#root__title) {
    display: none !important;
  }
  [data-slot="field-description"]#root__description {
    display: none !important;
  }
  form.flex.flex-col.gap-4 {
    gap: 28px !important;
  }
`;

const applyFormataShadowOverrides = (component, attempt = 0) => {
  const shadowRoot = component.shadowRoot;
  if (!shadowRoot) {
    if (attempt < 10) {
      window.requestAnimationFrame(() => {
        applyFormataShadowOverrides(component, attempt + 1);
      });
    }
    return;
  }

  let style = shadowRoot.querySelector("style[data-attesta-formata-overrides]");
  if (!(style instanceof HTMLStyleElement)) {
    style = document.createElement("style");
    style.dataset.attestaFormataOverrides = "true";
    shadowRoot.appendChild(style);
  }

  style.textContent = formataShadowOverrides;
};

const getPreferredTheme = () => {
  try {
    const stored = localStorage.getItem(themeStorageKey);
    if (stored === "light" || stored === "dark") {
      return stored;
    }
  } catch (err) {}

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
    const nextThemeLabel = theme === "dark" ? "light" : "dark";
    themeToggle.setAttribute("aria-label", `Switch to ${nextThemeLabel} theme`);
    themeToggle.setAttribute("title", `Switch to ${nextThemeLabel} theme`);
  }
};

const setTheme = (theme) => {
  applyTheme(theme);
  try {
    localStorage.setItem(themeStorageKey, theme);
  } catch (err) {}
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
const processPageContent = document.getElementById("process-page-content");
let skipNextProcessUpdatedEvent = false;
let skipNextProcessUpdatedEventTimer = 0;

const focusNextActionInput = () => {
  const container = processPageContent || document;
  const nextInput = container.querySelector(
    "input:not([disabled]):not([type='hidden']), textarea:not([disabled]), select:not([disabled])",
  );
  if (nextInput) {
    nextInput.focus();
  }
};

const syncSelectedSubstepURL = (substepId = "") => {
  try {
    const url = new URL(window.location.href);
    if (substepId) {
      url.searchParams.set("substep", substepId);
    } else {
      url.searchParams.delete("substep");
    }
    window.history.replaceState({}, "", url);
  } catch (_err) {}
};

const currentSelectedSubstep = () => {
  const selectedPanel = document.querySelector(
    ".js-process-substep-panel[open]",
  );
  if (selectedPanel instanceof HTMLElement) {
    return (selectedPanel.dataset.substepId || "").trim();
  }
  const selected = (root?.dataset?.selectedSubstep || "").trim();
  if (selected) {
    return selected;
  }
  return "";
};

const markSelectedSubstep = (substepId = "") => {
  const selected = substepId.trim();
  if (root instanceof HTMLElement) {
    root.dataset.selectedSubstep = selected;
  }
  for (const node of document.querySelectorAll("[data-substep-id]")) {
    if (!(node instanceof HTMLElement)) {
      continue;
    }
    const container = node.classList.contains("substep")
      ? node
      : node.closest(".substep");
    if (!(container instanceof HTMLElement)) {
      continue;
    }
    container.classList.toggle(
      "substep-selected",
      selected !== "" && node.dataset.substepId === selected,
    );
  }
  syncSelectedSubstepURL(selected);
};

const resolveAbsoluteURL = (value) => {
  try {
    return new URL(value, window.location.origin).toString();
  } catch (_err) {
    return value;
  }
};

const loadProcessContent = async (substepId = currentSelectedSubstep()) => {
  if (!processId || !workflowKey || !processPageContent) {
    return;
  }
  const query = substepId ? `?substep=${encodeURIComponent(substepId)}` : "";
  const url = `/w/${workflowKey}/process/${processId}/content${query}`;
  if (window.htmx?.ajax) {
    window.htmx.ajax("GET", url, {
      target: "#process-page-content",
      swap: "innerHTML",
    });
    return;
  }
  try {
    const response = await fetch(url);
    if (!response.ok) {
      return;
    }
    const html = await response.text();
    processPageContent.innerHTML = html;
    await initializeFormataForms(processPageContent);
    markSelectedSubstep(currentSelectedSubstep());
    focusNextActionInput();
  } catch (_err) {}
};

let substepOverrideModal;
let substepOverrideEditor;
let substepOverrideMessageHandler;

const ensureSubstepOverrideModal = () => {
  if (substepOverrideModal instanceof HTMLDialogElement) {
    return substepOverrideModal;
  }
  const dialog = document.createElement("dialog");
  dialog.className = "substep-override-modal";
  dialog.setAttribute("role", "dialog");
  dialog.setAttribute("aria-modal", "true");
  dialog.addEventListener("close", () => {
    if (substepOverrideMessageHandler) {
      window.removeEventListener("message", substepOverrideMessageHandler);
      substepOverrideMessageHandler = undefined;
    }
    dialog.innerHTML = "";
    substepOverrideEditor = undefined;
  });
  document.body.appendChild(dialog);
  substepOverrideModal = dialog;
  return dialog;
};

const closeSubstepOverrideModal = () => {
  if (substepOverrideMessageHandler) {
    window.removeEventListener("message", substepOverrideMessageHandler);
    substepOverrideMessageHandler = undefined;
  }
  if (
    substepOverrideModal instanceof HTMLDialogElement &&
    substepOverrideModal.open
  ) {
    substepOverrideModal.close();
  }
  if (substepOverrideModal instanceof HTMLDialogElement) {
    substepOverrideModal.innerHTML = "";
  }
  substepOverrideEditor = undefined;
};

const setSubstepOverrideError = (editor, message) => {
  const errorNode = editor?.querySelector(".js-substep-override-error");
  if (errorNode instanceof HTMLElement) {
    errorNode.textContent = message || "";
  }
};

const initializeSubstepOverrideEditor = (editor) => {
  if (!(editor instanceof HTMLElement)) {
    return;
  }
  substepOverrideEditor = editor;

  if (substepOverrideMessageHandler) {
    window.removeEventListener("message", substepOverrideMessageHandler);
    substepOverrideMessageHandler = undefined;
  }

  const iframe = editor.querySelector(".js-substep-override-frame");
  if (!(iframe instanceof HTMLIFrameElement)) {
    return;
  }

  const saveUrl = editor.dataset.saveUrl || "";
  if (!saveUrl) {
    setSubstepOverrideError(editor, "Save route is missing.");
    return;
  }

  let schema;
  let uiSchema;
  try {
    schema = JSON.parse(editor.dataset.schema || "{}");
    uiSchema = JSON.parse(editor.dataset.uischema || "{}");
  } catch (_err) {
    setSubstepOverrideError(editor, "Unable to load current schema.");
    return;
  }

  const builderBase = (
    editor.dataset.builderOrigin || window.location.origin
  ).replace(/\/$/, "");
  let builderOrigin = window.location.origin;
  try {
    builderOrigin = new URL(builderBase, window.location.origin).origin;
  } catch (_err) {}

  iframe.src =
    `${builderBase}/formata-arch/#/single-form` +
    `?targetOrigin=${encodeURIComponent(window.location.origin)}`;

  substepOverrideMessageHandler = async (event) => {
    if (event.origin !== builderOrigin) {
      return;
    }
    if (event.data?.type === "formata:schema-ready") {
      iframe.contentWindow?.postMessage(
        {
          type: "formata:schema-load",
          schema,
          uiSchema,
        },
        builderOrigin,
      );
      return;
    }
    if (event.data?.type !== "formata:schema-saved") {
      return;
    }
    const reason =
      typeof event.data.changeReason === "string"
        ? event.data.changeReason.trim()
        : "";
    if (!reason) {
      setSubstepOverrideError(editor, "Reason is required.");
      return;
    }
    try {
      const response = await fetch(saveUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify({
          schema: event.data.schema,
          uiSchema: event.data.uiSchema || {},
          changeReason: reason,
        }),
      });
      if (!response.ok) {
        const message = await response.text();
        setSubstepOverrideError(
          editor,
          message || "Failed to save local adaptation.",
        );
        return;
      }
      closeSubstepOverrideModal();
      await loadProcessContent(currentSelectedSubstep());
    } catch (_err) {
      setSubstepOverrideError(editor, "Failed to save local adaptation.");
    }
  };
  window.addEventListener("message", substepOverrideMessageHandler);
};

const openSubstepOverrideEditor = async (url) => {
  const dialog = ensureSubstepOverrideModal();
  dialog.innerHTML = '<div class="substep-override-loading">Loading...</div>';
  if (!dialog.open) {
    dialog.showModal();
  }
  try {
    const response = await fetch(url, { headers: { Accept: "text/html" } });
    const html = await response.text();
    if (!response.ok) {
      dialog.innerHTML =
        '<div class="substep-override-editor"><button type="button" class="secondary js-close-substep-override">Close</button><div class="error">Unable to open editor.</div></div>';
      return;
    }
    dialog.innerHTML = html;
    initializeSubstepOverrideEditor(
      dialog.querySelector(".js-substep-override-editor"),
    );
  } catch (_err) {
    dialog.innerHTML =
      '<div class="substep-override-editor"><button type="button" class="secondary js-close-substep-override">Close</button><div class="error">Unable to open editor.</div></div>';
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
      if (
        err &&
        typeof err === "object" &&
        "name" in err &&
        err.name === "AbortError"
      ) {
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
    } catch (_err) {}
  }
  window.prompt(`Copy ${shareLabel}:`, absoluteURL);
};

const triggerDownload = (button) => {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }
  const rawURL = button.dataset.downloadUrl;
  if (!rawURL) {
    return;
  }
  const absoluteURL = resolveAbsoluteURL(rawURL);
  const link = document.createElement("a");
  link.href = absoluteURL;
  link.download = "";
  link.style.display = "none";
  document.body.appendChild(link);
  link.click();
  link.remove();
};

const updateAttachmentCarousel = (carousel, nextIndex) => {
  if (!(carousel instanceof HTMLElement)) {
    return;
  }
  const slides = Array.from(carousel.querySelectorAll("[data-carousel-slide]"));
  if (slides.length === 0) {
    return;
  }
  const maxIndex = slides.length - 1;
  const safeIndex = Math.min(Math.max(nextIndex, 0), maxIndex);
  const currentIndex =
    Number.parseInt(carousel.dataset.carouselIndex || "0", 10) || 0;
  const currentSlide = slides[currentIndex];
  const nextSlide = slides[safeIndex];
  const prefersReducedMotion = window.matchMedia?.(
    "(prefers-reduced-motion: reduce)",
  )?.matches;

  if (
    currentSlide instanceof HTMLElement &&
    nextSlide instanceof HTMLElement &&
    currentIndex !== safeIndex &&
    !prefersReducedMotion
  ) {
    const direction = safeIndex > currentIndex ? 1 : -1;
    nextSlide.hidden = false;
    currentSlide.animate(
      [
        { opacity: 1, transform: "translateX(0)" },
        { opacity: 0, transform: `translateX(${direction * -18}px)` },
      ],
      { duration: 220, easing: "ease" },
    );
    nextSlide.animate(
      [
        { opacity: 0, transform: `translateX(${direction * 18}px)` },
        { opacity: 1, transform: "translateX(0)" },
      ],
      { duration: 220, easing: "ease" },
    );
  }

  slides.forEach((slide, index) => {
    if (!(slide instanceof HTMLElement)) {
      return;
    }
    slide.hidden = index !== safeIndex;
  });

  const dots = Array.from(carousel.querySelectorAll("[data-carousel-dot]"));
  dots.forEach((dot, index) => {
    if (!(dot instanceof HTMLButtonElement)) {
      return;
    }
    const active = index === safeIndex;
    if (active) {
      dot.setAttribute("aria-current", "true");
    } else {
      dot.removeAttribute("aria-current");
    }
  });
  const prevButton = carousel.querySelector("[data-carousel-prev]");
  if (prevButton instanceof HTMLButtonElement) {
    prevButton.disabled = safeIndex === 0;
  }
  const nextButton = carousel.querySelector("[data-carousel-next]");
  if (nextButton instanceof HTMLButtonElement) {
    nextButton.disabled = safeIndex === maxIndex;
  }
  carousel.dataset.carouselIndex = String(safeIndex);
};

const clearAttachmentSwipe = (carousel) => {
  if (!(carousel instanceof HTMLElement)) {
    return;
  }
  delete carousel.dataset.touchStartX;
  delete carousel.dataset.touchStartY;
};

const copyLinkValue = async (button) => {
  if (!(button instanceof HTMLButtonElement)) {
    return;
  }
  const rawURL = button.dataset.copyLink;
  if (!rawURL) {
    return;
  }
  const absoluteURL = resolveAbsoluteURL(rawURL);
  const label = button.dataset.copyLabel || "link";
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(absoluteURL);
      const originalText = button.textContent || "Copy";
      button.textContent = "Copied";
      window.setTimeout(() => {
        button.textContent = originalText;
      }, 1200);
      return;
    } catch (_err) {}
  }
  window.prompt(`Copy ${label}:`, absoluteURL);
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
    } catch (_err) {}
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
  if (
    normalized.type === "object" &&
    (!normalized.properties || typeof normalized.properties !== "object")
  ) {
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
  if (
    typeof description === "string" &&
    description.trim() !== "" &&
    typeof options.description !== "string"
  ) {
    options.description = description;
  }
  if (
    typeof help === "string" &&
    help.trim() !== "" &&
    typeof options.help !== "string"
  ) {
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
  const source =
    uiSchema.properties && typeof uiSchema.properties === "object"
      ? uiSchema.properties
      : uiSchema;
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
    reader.addEventListener("load", () => resolve(reader.result), {
      once: true,
    });
    reader.addEventListener(
      "error",
      () => reject(new Error("failed to read file")),
      { once: true },
    );
    reader.readAsDataURL(file);
  });

const serializeFormataValue = async (value) => {
  if (value instanceof File) {
    try {
      const dataURL = await fileToDataURL(value);
      if (typeof dataURL === "string" && dataURL.startsWith("data:")) {
        return dataURL;
      }
    } catch (_err) {}
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
  const detail =
    componentEvent instanceof CustomEvent ? componentEvent.detail : undefined;
  if (detail instanceof FormData) {
    return await formDataToObject(detail);
  }
  if (
    detail &&
    typeof detail === "object" &&
    detail.formData instanceof FormData
  ) {
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
  "https://closerproject.github.io/formata-arch/formata-web-component.umd.js",
];

const waitForFormataDefinition = (timeoutMs) => {
  if (customElements.get("formata-form")) {
    return Promise.resolve(true);
  }
  return Promise.race([
    customElements.whenDefined("formata-form").then(() => true),
    new Promise((resolve) => setTimeout(() => resolve(false), timeoutMs)),
  ]);
};

const loadExternalScript = (url) =>
  new Promise((resolve) => {
    const existing = document.querySelector(
      `script[data-formata-src="${url}"]`,
    );
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
    new Promise((resolve) => setTimeout(() => resolve(false), 5000)),
  ]);
};

const syncFormataValue = (component, hiddenInput) => {
  hiddenInput.value = JSON.stringify(readFormataComponentValue(component));
};

const pendingActiveRoleSubmits = new WeakMap();

const activeRoleInputForForm = (form) => {
  const input = form.querySelector("input[data-active-role-input]");
  return input instanceof HTMLInputElement ? input : null;
};

const activeRoleDialogForForm = (form) => {
  const dialogID = (form.dataset.activeRoleDialog || "").trim();
  if (!dialogID) {
    return null;
  }
  const dialog = document.getElementById(dialogID);
  return dialog instanceof HTMLDialogElement ? dialog : null;
};

const requestActiveRoleChoice = (form, onSelected) => {
  const roleInput = activeRoleInputForForm(form);
  if (!roleInput || roleInput.value.trim()) {
    return false;
  }
  const dialog = activeRoleDialogForForm(form);
  if (!dialog) {
    return false;
  }
  const error = dialog.querySelector("[data-active-role-error]");
  if (error instanceof HTMLElement) {
    error.hidden = true;
  }
  pendingActiveRoleSubmits.set(dialog, onSelected);
  if (typeof dialog.showModal === "function") {
    dialog.showModal();
  } else {
    dialog.setAttribute("open", "");
  }
  const checked = dialog.querySelector(
    'input[data-active-role-option]:checked',
  );
  if (checked instanceof HTMLInputElement) {
    checked.focus();
  } else {
    const firstOption = dialog.querySelector("input[data-active-role-option]");
    if (firstOption instanceof HTMLInputElement) {
      firstOption.focus();
    }
  }
  return true;
};

const submitFormataPayload = (form, hiddenInput, payload) => {
  const roleInput = activeRoleInputForForm(form);
  if (
    roleInput &&
    !roleInput.value.trim() &&
    requestActiveRoleChoice(form, () => {
      if (!submitFormataPayload(form, hiddenInput, payload)) {
        form.dataset.formataSubmitState = "idle";
      }
    })
  ) {
    return true;
  }

  const state = form.dataset.formataSubmitState || "idle";
  if (state !== "idle") {
    return false;
  }
  const serialized = JSON.stringify(payload ?? {});
  hiddenInput.value = serialized;
  form.dataset.formataSubmitState = "inflight";

  const url =
    form.dataset.formataPost ||
    form.getAttribute("hx-post") ||
    form.getAttribute("action");
  const target = form.dataset.formataTarget || "#process-page-content";
  const htmxApi = window.htmx;
  if (
    url &&
    htmxApi &&
    typeof htmxApi.ajax === "function" &&
    document.querySelector(target)
  ) {
    if (target === "#process-page-content") {
      skipNextProcessUpdatedEvent = true;
      window.clearTimeout(skipNextProcessUpdatedEventTimer);
      skipNextProcessUpdatedEventTimer = window.setTimeout(() => {
        skipNextProcessUpdatedEvent = false;
      }, 2000);
    }
    const values = { value: serialized };
    if (roleInput && roleInput.value.trim()) {
      values.activeRole = roleInput.value.trim();
    }
    htmxApi.ajax("POST", url, {
      source: form,
      target,
      swap: "innerHTML",
      values,
    });
    return true;
  }
  if (!url) {
    form.dataset.formataSubmitState = "idle";
    return false;
  }
  HTMLFormElement.prototype.submit.call(form);
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

    const rawSchema = parseJsonAttribute(
      host.getAttribute("data-formata-schema"),
    );
    const schema = normalizeFormataSchema(rawSchema);
    const rawUiSchema = parseJsonAttribute(
      host.getAttribute("data-formata-uischema"),
    );
    const uiSchema = normalizeFormataUiSchema(rawUiSchema);
    if (!schema) {
      continue;
    }

    const component = document.createElement("formata-form");
    component.className = "js-formata-form";
    if (host.dataset.formataDisabled !== "true") {
      component.setAttribute("prevent-page-reload", "");
    }
    if (document.documentElement.dataset.theme === "dark") {
      component.setAttribute("dark-mode", "");
    }
    if (host.dataset.formataDisabled === "true") {
      component.setAttribute("disabled", "");
    }
    component.schema = schema;
    component.uiSchema = uiSchema;
    host.appendChild(component);
    applyFormataShadowOverrides(component);

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
        payload = await serializeFormataValue(
          readFormataComponentValue(component),
        );
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
        payload = await serializeFormataValue(
          readFormataComponentValue(component),
        );
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

if (processId && workflowKey && processPageContent) {
  const source = new EventSource(
    `/w/${workflowKey}/events?workflow=${workflowKey}&processId=${processId}`,
  );
  source.addEventListener("process-updated", () => {
    if (skipNextProcessUpdatedEvent) {
      skipNextProcessUpdatedEvent = false;
      window.clearTimeout(skipNextProcessUpdatedEventTimer);
      return;
    }
    void loadProcessContent();
  });
}

document.addEventListener("DOMContentLoaded", () => {
  void initializeFormataForms(document);
  markSelectedSubstep(currentSelectedSubstep());
  focusNextActionInput();
});

document.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof Element)) {
    return;
  }
  const openDropdowns = document.querySelectorAll(
    ".account-menu[open], .workflow-card-menu[open]",
  );
  for (const dropdown of openDropdowns) {
    if (!dropdown.contains(target)) {
      dropdown.removeAttribute("open");
    }
  }
});

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.target && event.target.id === "process-page-content") {
    void initializeFormataForms(event.target);
    markSelectedSubstep(currentSelectedSubstep());
    focusNextActionInput();
  }
});

document.body.addEventListener("toggle", (event) => {
  const target = event.target;
  if (!(target instanceof HTMLDetailsElement)) {
    return;
  }
  if (!target.classList.contains("js-process-substep-panel")) {
    return;
  }
  const substepID = (target.dataset.substepId || "").trim();
  if (!substepID) {
    return;
  }
  if (target.open) {
    for (const panel of document.querySelectorAll(
      ".js-process-substep-panel",
    )) {
      if (!(panel instanceof HTMLDetailsElement) || panel === target) {
        continue;
      }
      panel.open = false;
    }
    markSelectedSubstep(substepID);
    focusNextActionInput();
    return;
  }
  if (currentSelectedSubstep() === substepID) {
    markSelectedSubstep("");
  }
});

document.body.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof Element)) {
    return;
  }
  const cancelActiveRoleButton = target.closest(".js-cancel-active-role");
  if (cancelActiveRoleButton instanceof HTMLButtonElement) {
    event.preventDefault();
    const dialog = cancelActiveRoleButton.closest(".js-active-role-dialog");
    if (dialog instanceof HTMLDialogElement) {
      pendingActiveRoleSubmits.delete(dialog);
      dialog.close();
    }
    return;
  }
  const confirmActiveRoleButton = target.closest(".js-confirm-active-role");
  if (confirmActiveRoleButton instanceof HTMLButtonElement) {
    event.preventDefault();
    const dialog = confirmActiveRoleButton.closest(".js-active-role-dialog");
    if (!(dialog instanceof HTMLDialogElement)) {
      return;
    }
    const selected = dialog.querySelector(
      'input[data-active-role-option]:checked',
    );
    const error = dialog.querySelector("[data-active-role-error]");
    if (!(selected instanceof HTMLInputElement)) {
      if (error instanceof HTMLElement) {
        error.hidden = false;
      }
      return;
    }
    const formID = (dialog.dataset.activeRoleForm || "").trim();
    const form = formID ? document.getElementById(formID) : null;
    const roleInput =
      form instanceof HTMLFormElement ? activeRoleInputForForm(form) : null;
    if (!roleInput) {
      pendingActiveRoleSubmits.delete(dialog);
      dialog.close();
      return;
    }
    roleInput.value = selected.value;
    const submit = pendingActiveRoleSubmits.get(dialog);
    pendingActiveRoleSubmits.delete(dialog);
    dialog.close();
    submit?.();
    return;
  }
  const closeOverrideButton = target.closest(".js-close-substep-override");
  if (closeOverrideButton instanceof HTMLButtonElement) {
    event.preventDefault();
    closeSubstepOverrideModal();
    return;
  }
  const overrideButton = target.closest(".js-open-substep-override");
  if (overrideButton instanceof HTMLButtonElement) {
    event.preventDefault();
    const url = overrideButton.dataset.overrideUrl || "";
    if (url) {
      void openSubstepOverrideEditor(url);
    }
    return;
  }
  const prevCarouselButton = target.closest("[data-carousel-prev]");
  if (prevCarouselButton instanceof HTMLButtonElement) {
    event.preventDefault();
    const carousel = prevCarouselButton.closest(".js-attachment-carousel");
    if (carousel instanceof HTMLElement) {
      const current =
        Number.parseInt(carousel.dataset.carouselIndex || "0", 10) || 0;
      updateAttachmentCarousel(carousel, current - 1);
    }
    return;
  }
  const nextCarouselButton = target.closest("[data-carousel-next]");
  if (nextCarouselButton instanceof HTMLButtonElement) {
    event.preventDefault();
    const carousel = nextCarouselButton.closest(".js-attachment-carousel");
    if (carousel instanceof HTMLElement) {
      const current =
        Number.parseInt(carousel.dataset.carouselIndex || "0", 10) || 0;
      updateAttachmentCarousel(carousel, current + 1);
    }
    return;
  }
  const dotButton = target.closest("[data-carousel-dot]");
  if (dotButton instanceof HTMLButtonElement) {
    event.preventDefault();
    const carousel = dotButton.closest(".js-attachment-carousel");
    if (carousel instanceof HTMLElement) {
      const index =
        Number.parseInt(dotButton.dataset.carouselDot || "0", 10) || 0;
      updateAttachmentCarousel(carousel, index);
    }
    return;
  }
  const downloadButton = target.closest(".js-download-link");
  if (downloadButton instanceof HTMLButtonElement) {
    event.preventDefault();
    triggerDownload(downloadButton);
    return;
  }
  const copyLinkButton = target.closest(".js-copy-link");
  if (copyLinkButton instanceof HTMLButtonElement) {
    event.preventDefault();
    void copyLinkValue(copyLinkButton);
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

document.body.addEventListener(
  "touchstart",
  (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const swipeSurface = target.closest("[data-carousel-swipe]");
    if (!(swipeSurface instanceof HTMLElement)) {
      return;
    }
    const carousel = swipeSurface.closest(".js-attachment-carousel");
    if (!(carousel instanceof HTMLElement)) {
      return;
    }
    const touch = event.touches?.[0];
    if (!touch) {
      return;
    }
    carousel.dataset.touchStartX = String(touch.clientX);
    carousel.dataset.touchStartY = String(touch.clientY);
  },
  { passive: true },
);

document.body.addEventListener(
  "touchcancel",
  (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const carousel = target.closest(".js-attachment-carousel");
    if (!(carousel instanceof HTMLElement)) {
      return;
    }
    clearAttachmentSwipe(carousel);
  },
  { passive: true },
);

document.body.addEventListener(
  "touchend",
  (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const swipeSurface = target.closest("[data-carousel-swipe]");
    if (!(swipeSurface instanceof HTMLElement)) {
      return;
    }
    const carousel = swipeSurface.closest(".js-attachment-carousel");
    if (!(carousel instanceof HTMLElement)) {
      return;
    }
    const startX = Number.parseFloat(carousel.dataset.touchStartX || "");
    const startY = Number.parseFloat(carousel.dataset.touchStartY || "");
    clearAttachmentSwipe(carousel);
    if (!Number.isFinite(startX) || !Number.isFinite(startY)) {
      return;
    }
    const touch = event.changedTouches?.[0];
    if (!touch) {
      return;
    }
    const deltaX = touch.clientX - startX;
    const deltaY = touch.clientY - startY;
    if (Math.abs(deltaX) < 40 || Math.abs(deltaX) <= Math.abs(deltaY)) {
      return;
    }
    const current =
      Number.parseInt(carousel.dataset.carouselIndex || "0", 10) || 0;
    updateAttachmentCarousel(carousel, deltaX < 0 ? current + 1 : current - 1);
  },
  { passive: true },
);

const deptRoot = document.querySelector("[data-dept-role]");
if (deptRoot) {
  const role = deptRoot.dataset.deptRole;
  const deptWorkflowKey = deptRoot.dataset.workflowKey;
  const dashboard = document.getElementById("dept-dashboard");
  if (role && deptWorkflowKey && dashboard) {
    const source = new EventSource(
      `/w/${deptWorkflowKey}/events?workflow=${deptWorkflowKey}&role=${role}`,
    );
    source.addEventListener("role-updated", async () => {
      try {
        const response = await fetch(
          `/w/${deptWorkflowKey}/backoffice/${role}/partial`,
        );
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
