document.addEventListener("DOMContentLoaded", async () => {
	const result = await fetch("/get-all-user-shares");
	if (result.status !== 200) {
		window.location.href = "/";
		return;
	}

	const res = await result.json();
	if (res["error_type"] !== undefined) {
		window.location.href = "/";
		return;
	}

	for (let i = 0; i < res.length; i++) {
		const clicked = decodeChart(res[i].chart);
		const date = new Date(clicked[0]);

		const chart = document.createElement("div");

		const controls = document.createElement("div");
		controls.classList.add("chart-controls");

		const a = document.createElement("a");
		a.href = "/s/" + res[i].path;
		a.innerText = a.href;
		controls.appendChild(a);

		const removeButton = document.createElement("button");
		removeButton.addEventListener("click", async () => {
			const removePath = res[i].path;
			const result = await fetch("/remove-chart", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ path : removePath })
			});
			const resJSON = await result.json();
			chart.remove();
		});
		removeButton.innerText = "Delete Share";
		removeButton.classList.add("button");
		removeButton.classList.add("button-red");
		controls.appendChild(removeButton);

		chart.appendChild(controls);
		chart.appendChild(newChart(date.getFullYear(), clicked));
		document.getElementById("charts").appendChild(chart);
	}
});

function newChart(year, stored) {
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

	const res = document.createElement("div");
	res.classList.add("chart");
	res.append(...weeks);
	return res;
}

function decodeChart(enc) {
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
			if ((v & (1 << bit)) !== 0) {
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

function urlSafeBase64Decode(arr) {
	let tmp = arr + Array((4 - arr.length % 4) % 4 + 1).join('=');
	tmp = tmp.replace(/={1,2}$/, '').replace(/_/g, '/').replace(/-/g, '+');
	return decode(tmp);
}

// https://github.com/WebReflection/uint8-to-base64/blob/master/index.js
var fromCharCode = String.fromCharCode;
var encode = function encode(uint8array) {
	var output = [];

	for (var i = 0, length = uint8array.length; i < length; i++) {
		output.push(fromCharCode(uint8array[i]));
	}

	return btoa(output.join(''));
};

var asCharCode = function asCharCode(c) {
	return c.charCodeAt(0);
};

var decode = function decode(chars) {
	return Uint8Array.from(atob(chars), asCharCode);
};
