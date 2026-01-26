import "./styles.css";

const root = document.querySelector("[data-process-id]");
const processId = root?.dataset?.processId;
const timeline = document.getElementById("timeline");

const focusNextActionInput = () => {
  const nextInput = document.querySelector(
    ".action-card.action-available:not(.action-other) input:not([disabled])"
  );
  if (nextInput) {
    nextInput.focus();
  }
};

if (processId && timeline) {
  const source = new EventSource(`/events?processId=${processId}`);
  source.addEventListener("process-updated", async () => {
    try {
      const response = await fetch(`/process/${processId}/timeline`);
      if (!response.ok) {
        return;
      }
      const html = await response.text();
      timeline.innerHTML = html;
    } catch (err) {
      // keep UI responsive even if SSE fails
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
  const dashboard = document.getElementById("dept-dashboard");
  if (role && dashboard) {
    const source = new EventSource(`/events?role=${role}`);
    source.addEventListener("role-updated", async () => {
      try {
        const response = await fetch(`/backoffice/${role}/partial`);
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
