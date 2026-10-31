const { test } = require("node:test");
const assert = require("node:assert/strict");
const { digitsOnly, formatRupiahInput } = require("../static/js/rupiah-input.js");

test("formats grouped Indonesian rupiah input", () => {
  assert.equal(formatRupiahInput("20000"), "20.000");
  assert.equal(formatRupiahInput("1000000"), "1.000.000");
  assert.equal(formatRupiahInput("0"), "0");
});

test("accepts pasted punctuation while keeping only whole rupiah digits", () => {
  assert.equal(digitsOnly("Rp. 20.000"), "20000");
  assert.equal(formatRupiahInput("Rp. 20.000"), "20.000");
});

test("limits input to the supported 13 digit amount", () => {
  assert.equal(digitsOnly("123456789012345"), "1234567890123");
  assert.equal(formatRupiahInput("123456789012345"), "1.234.567.890.123");
});
