import "./styles.css";

const themeStorageKey = "attesta_theme";
const themeToggle = document.getElementById("theme-toggle");

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
    ".action-card.action-available:not(.action-other) input:not([disabled])"
  );
  if (nextInput) {
    nextInput.focus();
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
  focusNextActionInput();
});

document.body.addEventListener("htmx:afterSwap", (event) => {
  if (event.target && event.target.id === "action-area") {
    focusNextActionInput();
  }
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
