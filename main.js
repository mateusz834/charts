document.addEventListener("DOMContentLoaded", () => {
	const chart = {
		yearInput: document.getElementById("year"),
		chartResetButton : document.getElementById("reset"),
		chart: document.getElementById("chart"),
		cmd: document.getElementById("cmd"),
		commitMsgInput: document.getElementById("commit-name"),
		copyToClipboardButton: document.getElementById("copy-to-clipboard"),
		gitReproducer: document.getElementById("chart-git-reproducer"),

		shareModalOpenButton: document.getElementById("share-modal-open"),
		shareModal: document.getElementById("share-modal"),
		shareModalLink: document.getElementById("share-modal-link"),
		shareModalCloseButton: document.getElementById("share-modal-close"),
		shareModalCopyLinkButton: document.getElementById("share-modal-copy-link"),

		editSharedButton: document.getElementById("edit-shared"),

		preview: false,

		start: function() {
			let saved;

			const url = new URL(document.location.href);
			const encodedChart = url.searchParams.get("s");
			if (encodedChart != null) {
				saved = this.decodeChart(encodedChart);
				this.preview = true;
			} else {
				saved = this.getSavedChart();
			}

			let date = new Date();
			if (saved !== null && saved.length !== 0) {
				date = new Date(saved[0]);
			}

			this.yearInput.value = date.getFullYear();
			this.generateChart(date.getFullYear(), saved)

			const parseInteager = (str) => {
				let num = parseInt(str, 10);
				if (!Number.isFinite(num)) {
					num = 0;
				}
				return num
			}

			this.shareModalOpenButton.addEventListener("click", () => {
				this.shareModal.classList.remove("hidden");
			});

			this.shareModalCloseButton.addEventListener("click", () => {
				this.shareModal.classList.add("hidden");
			});

			this.shareModal.addEventListener("click", (e) => {
				if (e.target === this.shareModal) {
					this.shareModal.classList.add("hidden");
				}
			});

			this.shareModalCopyLinkButton.addEventListener("click", () => {
				navigator.clipboard.writeText(this.shareModalLink.href);
			});

			this.yearInput.addEventListener("input", () => this.generateChart(parseInteager(this.yearInput.value), null));
			this.chartResetButton.addEventListener("click", () => this.generateChart(parseInteager(this.yearInput.value), null));
			this.commitMsgInput.addEventListener("input", () => this.chartUpdate());
			this.copyToClipboardButton.addEventListener("click", () => navigator.clipboard.writeText(cmd.innerText));

			this.editSharedButton.addEventListener("click", () => {
				this.editSharedButton.classList.add("hidden");
				this.chartResetButton.classList.remove("hidden");
				this.yearInput.removeAttribute("disabled");;
				this.preview = false;

				// remove hash from url, so that on next refresh we get the chart from localStorage.
				const url = new URL(document.location.href);
				url.searchParams.delete("s");
				history.replaceState({}, null, url);

				// to update the localStorage.
				this.chartUpdate();
			});

			this.chart.addEventListener("click", (e) => {
				if (e.target.classList.contains("day")) {
					this.click(e.target);
				}
			});
			this.chart.addEventListener("mousedown", (e) => {
				if (e.button !== 0) {
					return;
				}
				e.preventDefault();
				this.down = true;
				this.downElement = e.target;
			});
			document.addEventListener("mouseup", (e) => {
				if (e.button !== 0) {
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
			if (!this.preview) {
				element.classList.toggle("clicked");
				this.chartUpdate();
			}
		},

		getSavedChart: function() {
			let stored = localStorage.getItem("clicked");
			if (stored !== null) {
				return JSON.parse(stored);
			}
			return null;
		},

		generateChart: function(year, stored) {
			if (year < 1000 || year > 0xffff) {
				this.chart.classList.add("hidden");
				this.gitReproducer.classList.add("hidden");
				this.chartResetButton.setAttribute("disabled", "");
				this.shareModalOpenButton.setAttribute("disabled", "");
				this.cmd.innerHTML = "";
				this.chart.innerHTML = "";
				return;
			}

			let date = new Date(year, 0, 0, 12);

			let weeks = [];
			let week = undefined;

			while (true) {
				date = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1, 12);
				if (date.getFullYear() !== year) {
					break;
				}

				if (date.getDay() === 0 || week === undefined) {
					week = document.createElement("div");
					week.classList.add("week");
					weeks.push(week);
				}

				if (date.getMonth() === 0 && date.getDate() === 1) {
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

			this.chartResetButton.removeAttribute("disabled");
			this.shareModalOpenButton.removeAttribute("disabled");
			this.chart.replaceChildren(...weeks);
			this.chart.classList.remove("hidden");
			this.gitReproducer.classList.add("hidden");
			this.cmd.innerHTML = "";
			this.chartUpdate();

			if (this.preview) {
				this.editSharedButton.classList.remove("hidden");
				this.chartResetButton.classList.add("hidden");
				this.yearInput.setAttribute("disabled", "");;
			}

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
				if (!this.preview) {
					localStorage.setItem("clicked", JSON.stringify(clicked));
				}
			}

			this.shareModalLink.href = "/?s=" + this.encodeChart();
			this.shareModalLink.innerText = this.shareModalLink.href;
		},

		encodeChart: function() {
			const arr = new Uint8Array(2 + 53);
			var lastNonZero = 0;
			this.chart.querySelectorAll(".day").forEach((node, index) => {
				const date = new Date(Date.parse(node.dataset.date));
				if (index == 0) {
					const year = date.getFullYear();
					// Encode year as a 16b inteager in big-endian form.
					arr[0] = (year >> 8) & 0xff
					arr[1] = year & 0xff
				}

				const arrIndex = Math.floor(index / 8) + 2;

				if (node.classList.contains("clicked")) {
					if (arrIndex > lastNonZero) {
						lastNonZero = arrIndex
					}
					const bitIndex = index % 8;
					arr[arrIndex] |= 1 << bitIndex;
				} else {
					return
				}

			});

			return "0" + urlSafeBase64Encode(arr.subarray(0, lastNonZero+1));
		},

		decodeChart: function(enc) {
			if (enc[0] !== '0') {
				throw new Error("invalid encoding");
			}

			const arr = urlSafeBase64Decode(enc.substring(1));

			if (arr.length < 2) {
				throw new Error("invalid encoding");
			}

			const year = (arr[0] << 8) | arr[1]

			const clicked = [];
			let lastZero = false;
			arr.slice(2).forEach((v, i) => {
				lastZero = false;
				if (v == 0) {
					lastZero = true;
				}

				for (let bit = 7; bit >= 0; bit--) {
					if ((v & (1 << (7 - bit))) !== 0) {
						const dayNum = 1 + i*8 + (7 - bit);
						const date = new Date(year, 0, dayNum, 12);
						if (date.getFullYear() !== year) {
							throw new Error("invalid encoding");
						}
						clicked.push(date.getTime());
					}
				}
			});

			if (lastZero) {
				throw new Error("invalid encoding");
			}

			return clicked;
		}
	}

	chart.start();
});

function urlSafeBase64Encode(arr) {
	return btoa(arr).replace(/\//g, '_').replace(/\+/g, '-').replace(/={1,2}$/, '');
}

function urlSafeBase64Decode(arr) {
    const tmp = arr + Array((4 - arr.length % 4) % 4 + 1).join('=');
	tmp = tmp.replace(/={1,2}$/, '').replace(/_/g, '/').replace(/-/g, '+');
	return JSON.parse("[" + atob(tmp) + "]"); // heh
}

