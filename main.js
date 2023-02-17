document.addEventListener("DOMContentLoaded", () => {
	var chart = {
		yearInput: document.getElementById("year"),
		chartResetButton : document.getElementById("reset"),
		chart: document.getElementById("chart"),
		cmd: document.getElementById("cmd"),
		commitMsgInput: document.getElementById("commit-name"),
		copyToClipboardButton: document.getElementById("copy-to-clipboard"),
		gitReproducer: document.getElementById("chart-git-reproducer"),

		start: function() {
			let date = new Date();
			let saved = this.getSavedChart();
			if (saved != null && saved.length != 0) {
				date = new Date(saved[0]);
			}

			this.yearInput.value = date.getFullYear();
			this.generateChart(date.getFullYear(), saved)

			this.yearInput.addEventListener("input", () => this.generateChart(this.yearInput.value, null));
			this.chartResetButton.addEventListener("click", () => this.generateChart(this.yearInput.value, null));
			this.commitMsgInput.addEventListener("input", () => this.chartUpdate());
			this.copyToClipboardButton.addEventListener("click", () => navigator.clipboard.writeText(cmd.innerText));

			this.chart.addEventListener("click", (e) => {
				if (e.target.classList.contains("day")) {
					this.click(e.target);
				}
			});
			this.chart.addEventListener("mousedown", (e) => {
				if (e.button != 0) {
					return;
				}
				e.preventDefault();
				this.down = true;
				this.downElement = e.target;
			});
			document.addEventListener("mouseup", (e) => {
				if (e.button != 0) {
					return;
				}
				this.down = false;
			});
			this.chart.addEventListener("mousemove", (e) => {
				if (!this.down) {
					return;
				}

				if (this.downElement !== null && e.target !== this.downElement) {
					const element = this.downElement;
					this.downElement = null;

					if (element.classList.contains("day")) {
						this.clickWithBlock(element);
					}
				}

				if (e.target.classList.contains("day")) {
					if (e.target.classList.contains("non-clickable") || e.target === this.downElement) {
						return;
					}
					this.clickWithBlock(e.target);
				}
			});
		},

		clickWithBlock: function(element) {
			this.click(element);
			element.classList.add("non-clickable");
			setTimeout(() => element.classList.remove("non-clickable"), 300);
		},

		click: function(element) {
			element.classList.toggle("clicked");
			this.chartUpdate();
		},

		getSavedChart: function() {
			let stored = localStorage.getItem("clicked");
			if (stored != null) {
				return JSON.parse(stored);
			}
			return null;
		},

		generateChart: function(year, stored) {
			if (this.yearInput.value.length < 4) {
				this.chart.classList.add("hidden");
				this.chart.innerHTML = "";
				return;
			}

			let date = new Date(year, 0, 0);

			let weeks = [];
			let week = undefined;

			while (true) {
				date = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
				if (date.getFullYear() != year) {
					break;
				}

				if (date.getDay() == 0 || week == undefined) {
					week = document.createElement("div");
					week.classList.add("week");
					weeks.push(week);
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

			this.chart.replaceChildren(...weeks);
			this.chart.classList.remove("hidden");
			this.gitReproducer.classList.add("hidden");
			this.cmd.innerHTML = "";
			this.chartUpdate();
		},

		chartUpdate: function() {
			const clicked = [];
			let cmds = "";
			this.chart.querySelectorAll(".clicked").forEach((node, index) => {
				if (cmds !== "") {
					cmds += "\n" + "git commit --date \"" + node.dataset.date + "\" -m \"" + this.commitMsgInput.value + "\""
				} else {
					cmds = "git commit --date \"" + node.dataset.date + "\" -m \"" + this.commitMsgInput.value + "\""
				}
				clicked[index] = Date.parse(node.dataset.date);
			});

			if (clicked.length !== 0) {
				this.cmd.innerText = cmds;
				this.gitReproducer.classList.remove("hidden");
				localStorage.setItem("clicked", JSON.stringify(clicked));
			}
		}
	}
	chart.start();
});


