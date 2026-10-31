const toastRegion = document.createElement("div");
const notificationKeys = new Set();
toastRegion.id = "app-toast-region";
toastRegion.setAttribute("aria-label", "Notifikasi");
toastRegion.setAttribute("aria-live", "polite");
toastRegion.setAttribute("aria-relevant", "additions");

function placeToastRegion() {
  const openDialogs = [...document.querySelectorAll("dialog[open]")];
  const topDialog = openDialogs.at(-1);
  (topDialog || document.body).append(toastRegion);
}

function notificationLevel(source) {
  const classes = source.className?.toString() || "";
  if (/amber|warning/i.test(classes)) return "warning";
  if (/green|success/i.test(classes)) return "success";
  if (/blue|info/i.test(classes)) return "info";
  return source.getAttribute("role") === "alert" ? "error" : "info";
}

function makeNotificationIcon(level) {
  const paths = {
    success: '<circle cx="12" cy="12" r="9"/><path d="m8 12 2.5 2.5L16 9"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5m0-8h.01"/>',
    warning: '<path d="M10.3 3.9 2.6 17.2A2 2 0 0 0 4.3 20h15.4a2 2 0 0 0 1.7-2.8L13.7 3.9a2 2 0 0 0-3.4 0Z"/><path d="M12 9v4m0 3h.01"/>',
    error: '<path d="M10.3 3.9 2.6 17.2A2 2 0 0 0 4.3 20h15.4a2 2 0 0 0 1.7-2.8L13.7 3.9a2 2 0 0 0-3.4 0Z"/><path d="m9.5 9.5 5 5m0-5-5 5"/>',
  };
  const icon = document.createElement("span");
  icon.className = `app-toast-icon app-toast-icon-${level}`;
  icon.setAttribute("aria-hidden", "true");
  icon.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">${paths[level]}</svg>`;
  return icon;
}

function showNotification(source) {
  const level = notificationLevel(source);
  const key = `${level}:${source.textContent.trim().replace(/\s+/g, " ")}`;
  if (notificationKeys.has(key)) {
    source.remove();
    return;
  }
  notificationKeys.add(key);

  const toast = document.createElement("div");
  toast.className = `app-toast app-toast-${level}`;
  toast.setAttribute("role", level === "error" || level === "warning" ? "alert" : "status");

  const content = document.createElement("div");
  content.className = "app-toast-message";
  while (source.firstChild) content.append(source.firstChild);

  const close = document.createElement("button");
  close.className = "app-toast-close";
  close.type = "button";
  close.setAttribute("aria-label", "Tutup notifikasi");
  close.dataset.toastClose = "";
  close.innerHTML = '<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><path d="m5 5 10 10M15 5 5 15"/></svg>';

  toast.append(makeNotificationIcon(level), content, close);
  source.remove();
  toastRegion.append(toast);
  window.setTimeout(() => toast.remove(), 5000);
}

document.body.append(toastRegion);
document.querySelectorAll('[role="alert"], [role="status"]:not([data-instant-status])').forEach(showNotification);
document.querySelectorAll("[data-credential-reveal]").forEach((reveal) => {
  window.setTimeout(() => reveal.remove(), 5 * 60 * 1000);
});
placeToastRegion();

function closeRowActionMenus(except = null) {
  document.querySelectorAll("[data-row-action-menu][open]").forEach((menu) => {
    if (menu !== except) menu.open = false;
  });
}

function positionRowActionMenu(menu) {
  const trigger = menu.querySelector("[data-row-action-toggle]");
  const panel = menu.querySelector(".row-action-panel");
  if (!trigger || !panel || !menu.open) return;

  const triggerRect = trigger.getBoundingClientRect();
  const panelRect = panel.getBoundingClientRect();
  const gap = 8;
  const margin = 8;
  const top = triggerRect.bottom + panelRect.height + margin <= window.innerHeight
    ? triggerRect.bottom + gap
    : Math.max(margin, triggerRect.top - panelRect.height - gap);
  const left = Math.max(margin, Math.min(triggerRect.right - panelRect.width, window.innerWidth - panelRect.width - margin));
  panel.style.top = `${top}px`;
  panel.style.left = `${left}px`;
}

document.addEventListener("click", (event) => {
  const rowActionToggle = event.target.closest("[data-row-action-toggle]");
  if (rowActionToggle) {
    const menu = rowActionToggle.closest("[data-row-action-menu]");
    closeRowActionMenus(menu);
    window.requestAnimationFrame(() => positionRowActionMenu(menu));
    return;
  }

  const rowActionItem = event.target.closest("[data-row-action-item]");
  if (rowActionItem) {
    rowActionItem.closest("[data-row-action-menu]").open = false;
  }

  if (!event.target.closest("[data-row-action-menu]")) closeRowActionMenus();

  const opener = event.target.closest("[data-modal-open]");
  if (opener) {
    const modal = document.getElementById(opener.dataset.modalOpen);
    modal?.showModal();
    if (modal) placeToastRegion();
    return;
  }

  const toastCloser = event.target.closest("[data-toast-close]");
  if (toastCloser) {
    toastCloser.closest(".app-toast")?.remove();
    return;
  }

  const credentialCloser = event.target.closest("[data-credential-dismiss]");
  if (credentialCloser) {
    credentialCloser.closest("[data-credential-reveal]")?.remove();
    return;
  }

  const copyButton = event.target.closest("[data-copy-value]");
  if (copyButton) {
    const originalLabel = copyButton.textContent;
    const value = copyButton.dataset.copyValue;
    const fallbackCopy = () => {
      const input = document.createElement("textarea");
      input.value = value;
      input.setAttribute("readonly", "");
      input.style.position = "fixed";
      input.style.opacity = "0";
      document.body.append(input);
      input.select();
      const copied = document.execCommand("copy");
      input.remove();
      return copied ? Promise.resolve() : Promise.reject(new Error("copy failed"));
    };
    const copy = navigator.clipboard?.writeText
      ? navigator.clipboard.writeText(value)
      : fallbackCopy();
    copy.then(() => {
      copyButton.textContent = "Tersalin";
      window.setTimeout(() => { if (copyButton.isConnected) copyButton.textContent = originalLabel; }, 1600);
    }).catch(() => {
      copyButton.textContent = "Gagal menyalin";
      window.setTimeout(() => { if (copyButton.isConnected) copyButton.textContent = originalLabel; }, 1600);
    });
    return;
  }

  const closer = event.target.closest("[data-modal-close]");
  if (closer) {
    closer.closest("dialog")?.close();
  }

  if (event.target instanceof HTMLDialogElement && event.target.classList.contains("app-modal")) {
    event.target.close();
  }
});

document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape") return;
  const openMenu = document.querySelector("[data-row-action-menu][open]");
  if (!openMenu) return;
  openMenu.open = false;
  openMenu.querySelector("[data-row-action-toggle]")?.focus();
});

window.addEventListener("scroll", () => closeRowActionMenus(), true);
window.addEventListener("resize", () => closeRowActionMenus());

document.addEventListener("close", (event) => {
  if (event.target instanceof HTMLDialogElement && event.target.contains(toastRegion)) {
    placeToastRegion();
  }
}, true);

document.querySelectorAll("dialog[data-modal-show]").forEach((modal) => modal.showModal());
placeToastRegion();

document.querySelectorAll("[data-user-role-form]").forEach((form) => {
  const role = form.querySelector("select[name='role']");
  const updateFields = () => {
    form.querySelectorAll("[data-role-fields]").forEach((group) => {
      const active = group.dataset.roleFields === role.value;
      group.hidden = !active;
      group.querySelectorAll("input, select").forEach((input) => {
        input.required = active;
      });
    });
  };
  role.addEventListener("change", updateFields);
  updateFields();
});

document.querySelectorAll("[data-student-class-filter]").forEach((group) => {
  const classSelect = group.querySelector("[data-student-class]");
  const studentSelect = group.querySelector("[data-student-options]");
  if (!classSelect || !studentSelect) return;
  let currentRequest;

  const loadStudents = async () => {
    const classId = classSelect.value;
    const selectedStudent = group.dataset.selectedStudent || "";
    currentRequest?.abort();
    studentSelect.replaceChildren(new Option(classId ? "Memuat siswa…" : "Pilih kelas terlebih dahulu", ""));
    studentSelect.disabled = true;
    if (!classId) return;

    try {
      currentRequest = new AbortController();
      const response = await fetch(`/users/student-options?class_id=${encodeURIComponent(classId)}`, {
        headers: { Accept: "application/json" },
        credentials: "same-origin",
        signal: currentRequest.signal,
      });
      if (!response.ok) throw new Error("student options unavailable");
      const students = await response.json();
      if (classSelect.value !== classId) return;
      studentSelect.replaceChildren(new Option(students.length ? "Pilih siswa" : "Tidak ada siswa tanpa akun di kelas ini", ""));
      students.forEach((student) => studentSelect.add(new Option(student.Label, student.ID)));
      studentSelect.disabled = students.length === 0;
      if (students.some((student) => student.ID === selectedStudent)) studentSelect.value = selectedStudent;
      group.dataset.selectedStudent = "";
    } catch (error) {
      if (error.name === "AbortError") return;
      studentSelect.replaceChildren(new Option("Daftar siswa gagal dimuat", ""));
    }
  };

  classSelect.addEventListener("change", loadStudents);
  if (classSelect.value) loadStudents();
});

document.addEventListener("submit", (event) => {
  const message = event.target.dataset.confirm;
  if (message && !window.confirm(message)) {
    event.preventDefault();
  }
});

document.querySelectorAll('[data-student-placement-form]').forEach((form) => {
  const year = form.elements.academic_year_id;
  const classes = form.elements.class_id;
  const help = form.querySelector('[data-placement-help]');
  let request;
  const showDate = () => {
    const date = classes.selectedOptions[0]?.dataset.effectiveDate;
    help.textContent = date
      ? `Tanggal efektif: ${date}, sampai akhir tahun ajaran. Akun login dapat dibuat melalui menu Pengguna.`
      : 'Pilih kelas yang sudah dikonfigurasi pada tahun ajaran ini. Akun login dapat dibuat melalui menu Pengguna.';
  };
  classes.addEventListener('change', showDate);
  year.addEventListener('change', async () => {
    request?.abort();
    request = new AbortController();
    classes.replaceChildren(new Option('Pilih kelas', ''));
    classes.disabled = true;
    form.querySelector('button[type="submit"]').disabled = true;
    if (!year.value) {
      help.textContent = 'Pilih tahun ajaran terlebih dahulu.';
      form.querySelector('button[type="submit"]').disabled = false;
      return;
    }
    help.textContent = 'Memuat pilihan kelas…';
    try {
      const response = await fetch(`/students/class-options?academic_year_id=${encodeURIComponent(year.value)}`, { signal: request.signal });
      if (!response.ok || !response.headers.get('Content-Type')?.includes('application/json')) throw new Error('load');
      const options = await response.json();
      for (const item of options) {
        const option = new Option(item.name, item.id);
        option.dataset.effectiveDate = item.effective_date;
        classes.append(option);
      }
      classes.disabled = false;
      form.querySelector('button[type="submit"]').disabled = false;
      showDate();
      if (!options.length) help.textContent = 'Belum ada kelas untuk tahun ajaran ini. Tambahkan kelas melalui menu Kelas.';
    } catch (error) {
      if (error.name !== 'AbortError') help.textContent = 'Pilihan kelas gagal dimuat. Pilih ulang tahun ajaran untuk mencoba lagi.';
    }
  });
  showDate();
});
