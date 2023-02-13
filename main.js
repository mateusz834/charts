function update(year, stored) {
	const chart = document.getElementById("chart");
	const el = document.createElement("div");
	el.id = "chart";

	el.addEventListener("click", (e) => click(e.target));

	var down = false;
	var downElement = null;
	el.addEventListener("mousedown", (e) => {
		if (e.button != 0) {
			return
		}
		down = true;
		downElement = e.target;
	});
	el.addEventListener("mouseup", (e) => {
		if (e.button != 0) {
			return
		}
		down = false;
	});
	el.addEventListener("mousemove", (e) => {
		if (!down) {
			return
		}

		if (downElement != null && e.target != downElement) {
			const element = downElement;
			downElement = null;

			if (element.classList.contains("day")) {
				click(element);
				element.classList.add("non-clickable");
				setTimeout(() => element.classList.remove("non-clickable"), 500);
			}
		}

		if (e.target.classList.contains("day")) {
			if (e.target.classList.contains("non-clickable") || e.target == downElement) {
				return
			}

			click(e.target);
			e.target.classList.add("non-clickable");
			setTimeout(() => e.target.classList.remove("non-clickable"), 500);
		}
	});

	let date = new Date(year, 0, 0);
	let week = undefined;

	while (true) {
		date = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
		if (date.getFullYear() != year) {
			break;
		}

		if (date.getDay() == 0 || week == undefined) {
			week = document.createElement("div");
			week.classList.add("week");
			el.appendChild(week);
		}

		if (date.getMonth() == 0 && date.getDate() == 1) {
			const day = date.getDay();
			for (let i = 0; i < day; i++) {
				const day = document.createElement("div");
				day.classList.add("no-day");
				week.appendChild(day);
			}
		}

		const day = document.createElement("div");
		day.classList.add("day");
		if (stored != null && stored.includes(date.getTime())) {
			day.classList.add("clicked");
		}
		day.dataset.date = date.toISOString();
		week.appendChild(day);
	}

	chart.replaceWith(el);
	generate_git_cmds();
}

function click(target) {
	if (!target.classList.contains("day")) {
		return;
	}
	target.classList.toggle("clicked");
	generate_git_cmds()
}

function generate_git_cmds() {
	const chart = document.getElementById("chart")
	const cmd = document.createElement("code")
	cmd.id = "cmd";

	const clicked = [];
	chart.querySelectorAll(".clicked").forEach((node, index) => {
		if (cmd.textContent !== "") {
			cmd.textContent += "\n" + "git commit --date \"" + node.dataset.date + "\" -m \"charts\""
		} else {
			cmd.textContent = "git commit --date \"" + node.dataset.date + "\" -m \"charts\""
		}
		const date = Date.parse(node.dataset.date);
		clicked[index] = date;
	});

	localStorage.setItem("clicked", JSON.stringify(clicked));
	document.getElementById("cmd").replaceWith(cmd);
}

document.addEventListener("DOMContentLoaded", () => {
	const year = document.getElementById("year");
	let date = new Date();

	let stored = localStorage.getItem("clicked");
	if (stored != null) {
		stored = JSON.parse(stored);
		if (stored.length != 0) {
			date = new Date(stored[0]);
		}
	}

	year.value = date.getFullYear();
	update(date.getFullYear(), stored)

	year.addEventListener("input", (y) => update(y.target.value));
	document.getElementById("reset").addEventListener("click", () => update(year.value, null));
});

