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
		const form = combo.closest("form");
		if (form) {
			// Tax ID and country are resolved from the database using this id.
			const id = form.querySelector('[name="cus_id"]');
			if (id) id.value = option.dataset.id || "";
		}
		setExpanded(combo, false);
		input.focus();
	}

	document.addEventListener("input", function (e) {
		const combo = comboOf(e.target);
		if (combo && e.target.classList.contains("combo__input")) openList(combo);
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
