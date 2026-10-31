(function (root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) module.exports = api;
  if (root.document) {
    const init = () => api.init(root.document);
    if (root.document.readyState === "loading") {
      root.document.addEventListener("DOMContentLoaded", init, { once: true });
    } else {
      init();
    }
  }
})(typeof globalThis !== "undefined" ? globalThis : this, function () {
  const MAX_DIGITS = 13;

  function digitsOnly(value) {
    return String(value).replace(/\D/g, "").slice(0, MAX_DIGITS);
  }

  function formatRupiahInput(value) {
    const digits = digitsOnly(value);
    return digits.replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  }

  function cursorAfterDigitCount(value, digitCount) {
    if (digitCount === 0) return 0;
    let seen = 0;
    for (let index = 0; index < value.length; index += 1) {
      if (/\d/.test(value[index])) seen += 1;
      if (seen === digitCount) return index + 1;
    }
    return value.length;
  }

  function init(document) {
    document.querySelectorAll("input[data-rupiah-input]").forEach((input) => {
      if (input.dataset.rupiahReady === "true") return;

      const originalName = input.name;
      const rawInput = document.createElement("input");
      rawInput.type = "hidden";
      rawInput.name = originalName;
      rawInput.value = digitsOnly(input.value);
      input.insertAdjacentElement("afterend", rawInput);

      input.removeAttribute("name");
      input.removeAttribute("min");
      input.removeAttribute("max");
      input.removeAttribute("step");
      input.type = "text";
      input.inputMode = "numeric";
      input.autocomplete = "off";
      input.maxLength = MAX_DIGITS + Math.floor((MAX_DIGITS - 1) / 3);
      input.placeholder = "20.000";
      input.dataset.rupiahReady = "true";

      const update = () => {
        const cursor = input.selectionStart ?? input.value.length;
        const digitsBeforeCursor = digitsOnly(input.value.slice(0, cursor)).length;
        const digits = digitsOnly(input.value);
        input.value = formatRupiahInput(digits);
        rawInput.value = digits;
        const nextCursor = cursorAfterDigitCount(input.value, digitsBeforeCursor);
        input.setSelectionRange(nextCursor, nextCursor);
      };

      input.addEventListener("input", update);
      input.form?.addEventListener("submit", update);
      input.value = formatRupiahInput(input.value);
      rawInput.value = digitsOnly(input.value);
    });
  }

  return { digitsOnly, formatRupiahInput, init };
});
