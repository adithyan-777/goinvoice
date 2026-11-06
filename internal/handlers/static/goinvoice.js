// goinvoice UI enhancements: a searchable combobox used by the document
// forms. Written with delegated listeners so it keeps working when forms are
// swapped into the modal via Datastar (no re-initialisation needed).

(function () {
	"use strict";

	function comboOf(el) {
		return el.closest ? el.closest("[data-combo]") : null;
	}

	function options(combo) {
		return Array.prototype.slice.call(combo.querySelectorAll(".combo__option"));
	}

	function setExpanded(combo, open) {
		const input = combo.querySelector(".combo__input");
		const list = combo.querySelector(".combo__list");
		if (!input || !list) return;
		list.hidden = !open;
		input.setAttribute("aria-expanded", open ? "true" : "false");
		combo.classList.toggle("is-open", open);
		if (!open) setActive(combo, -1);
	}

	function setActive(combo, index) {
		const opts = options(combo);
		opts.forEach(function (o, i) {
			o.classList.toggle("is-active", i === index);
		});
		combo.dataset.active = index >= 0 ? String(index) : "";
		if (index >= 0 && opts[index]) {
			opts[index].scrollIntoView({ block: "nearest" });
		}
	}

	function activeIndex(combo) {
		const raw = parseInt(combo.dataset.active || "-1", 10);
		return Number.isNaN(raw) ? -1 : raw;
	}

	function filter(combo) {
		const input = combo.querySelector(".combo__input");
		const empty = combo.querySelector(".combo__empty");
		const q = (input ? input.value : "").trim().toLowerCase();
		const opts = options(combo);
		let visible = 0;
		opts.forEach(function (o) {
			const hay = (o.dataset.value + " " + (o.dataset.code || "")).toLowerCase();
			const match = q === "" || hay.indexOf(q) !== -1;
			o.hidden = !match;
			if (match) visible++;
		});
		if (empty) empty.hidden = visible > 0;
		setActive(combo, -1);
	}

	function openList(combo) {
		filter(combo);
		setExpanded(combo, true);
	}

	function showAll(combo) {
		options(combo).forEach(function (o) { o.hidden = false; });
		const empty = combo.querySelector(".combo__empty");
		if (empty) empty.hidden = true;
		setActive(combo, -1);
		setExpanded(combo, true);
	}

	function choose(combo, option) {
		const input = combo.querySelector(".combo__input");
		if (!input || !option) return;
		input.value = option.dataset.value || option.textContent.trim();
		// Tax ID and country are resolved from the database using this id.
		// The hidden id field lives inside the same combo, named cus_id or
		// sup_id depending on the party picker.
		const id = combo.querySelector('input[type="hidden"]');
		if (id) id.value = option.dataset.id || "";
		// Product lines: autofill the unit price from the catalog. The user
		// can still edit it afterwards; the submitted value wins.
		if (combo.hasAttribute("data-product-combo") && option.dataset.price) {
			const line = combo.closest("[data-line]");
			const price = line ? line.querySelector("[data-price-input]") : null;
			if (price) price.value = option.dataset.price;
		}
		const lines = combo.closest("[data-lines]");
		setExpanded(combo, false);
		input.focus();
		if (lines) ensureDocLine(lines);
	}

	// Dynamic document lines: the form starts with one product row; typing
	// or picking a product in the last row appends a fresh empty row from
	// the hidden <template>.
	function appendDocLine(container, idx) {
		const form = container.closest("form");
		const tpl = form
			? form.querySelector("template[data-line-template]")
			: container.querySelector("template[data-line-template]");
		if (!tpl) return;
		const n = container.querySelectorAll("[data-line]").length;
		container.insertAdjacentHTML("beforeend", tpl.innerHTML.split("__IDX__").join(String(idx)));
		const rows = container.querySelectorAll("[data-line]");
		const title = rows[rows.length - 1].querySelector("[data-line-title]");
		if (title) title.textContent = "Line " + (n + 1);
	}

	function ensureDocLine(container) {
		const rows = container.querySelectorAll("[data-line]");
		if (!rows.length) {
			appendDocLine(container, 0);
			container.setAttribute("data-next-index", "1");
			return;
		}
		const input = rows[rows.length - 1].querySelector(".combo__input");
		if (input && input.value.trim() !== "") {
			let idx = parseInt(container.getAttribute("data-next-index") || "", 10);
			if (Number.isNaN(idx)) idx = rows.length;
			appendDocLine(container, idx);
			container.setAttribute("data-next-index", String(idx + 1));
		}
	}

	function renumberDocLines(container) {
		container.querySelectorAll("[data-line]").forEach(function (row, i) {
			const title = row.querySelector("[data-line-title]");
			if (title) title.textContent = "Line " + (i + 1);
		});
	}

	document.addEventListener("input", function (e) {
		const combo = comboOf(e.target);
		if (combo && e.target.classList.contains("combo__input")) {
			openList(combo);
			// Typing a product into the last line row grows a new row.
			const lines = combo.closest("[data-lines]");
			if (lines) ensureDocLine(lines);
		}
	});

	document.addEventListener("click", function (e) {
		// Clicking the input (its current value, e.g. the selected customer
		// name) toggles the list: open when closed, close when already open.
		const input = e.target.closest(".combo__input");
		if (input) {
			const combo = input.closest("[data-combo]");
			const list = combo.querySelector(".combo__list");
			if (list.hidden) {
				showAll(combo);
			} else {
				setExpanded(combo, false);
			}
			return;
		}
		const toggle = e.target.closest(".combo__toggle");
		if (toggle) {
			const combo = toggle.closest("[data-combo]");
			const list = combo.querySelector(".combo__list");
			if (list.hidden) {
				showAll(combo);
				combo.querySelector(".combo__input").focus();
			} else {
				setExpanded(combo, false);
			}
			return;
		}
		const option = e.target.closest(".combo__option");
		if (option) {
			choose(comboOf(option), option);
			return;
		}
		// Clicking outside closes any open combobox.
		if (!e.target.closest("[data-combo]")) {
			document.querySelectorAll("[data-combo].is-open").forEach(function (c) {
				setExpanded(c, false);
			});
		}
	});

	// Removing a document line row. The last remaining row is cleared
	// instead of removed so the form always submits at least one row.
	document.addEventListener("click", function (e) {
		const btn = e.target.closest("[data-remove-line]");
		if (!btn) return;
		const row = btn.closest("[data-line]");
		const container = btn.closest("[data-lines]");
		if (!row || !container) return;
		if (container.querySelectorAll("[data-line]").length <= 1) {
			const input = row.querySelector(".combo__input");
			if (input) input.value = "";
			const hid = row.querySelector('input[type="hidden"]');
			if (hid) hid.value = "";
			const price = row.querySelector("[data-price-input]");
			if (price) price.value = "";
			return;
		}
		row.remove();
		renumberDocLines(container);
	});

	document.addEventListener("keydown", function (e) {
		const combo = comboOf(e.target);
		if (!combo || !e.target.classList.contains("combo__input")) return;
		const list = combo.querySelector(".combo__list");
		const isOpen = list && !list.hidden;
		const opts = options(combo).filter(function (o) { return !o.hidden; });

		switch (e.key) {
		case "ArrowDown":
			e.preventDefault();
			if (!isOpen) { openList(combo); return; }
			setActive(combo, Math.min(activeIndex(combo) + 1, opts.length - 1));
			return;
		case "ArrowUp":
			e.preventDefault();
			if (!isOpen) { openList(combo); return; }
			setActive(combo, Math.max(activeIndex(combo) - 1, 0));
			return;
		case "Enter":
			if (isOpen && activeIndex(combo) >= 0 && opts[activeIndex(combo)]) {
				e.preventDefault();
				choose(combo, opts[activeIndex(combo)]);
			}
			return;
		case "Escape":
			if (isOpen) {
				e.preventDefault();
				e.stopPropagation();
				setExpanded(combo, false);
			}
			return;
		}
	});

	// Close open comboboxes when the modal/forms are replaced.
	document.addEventListener("datastar-fetch", function () {
		document.querySelectorAll("[data-combo].is-open").forEach(function (c) {
			setExpanded(c, false);
		});
	});
})();
