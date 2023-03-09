document.addEventListener("DOMContentLoaded", async () => {
	const path = document.location.pathname.substring(3);
	const result = await fetch("/share/" + path);
	if (result.status !== 200) {
		document.location.href = "/";
		return;
	}

	const res = await result.json();

	const clicked = decodeChart(res["chart"]);
	const date = new Date(clicked[0]);
	const chart = newChart(date.getFullYear(), clicked);
	chart.id = "chart-share-chart";


	const chartControls = document.createElement("div");
	chartControls.id = "chart-share-controls";

	const year = document.createElement("div");
	year.innerText = "Year: " + date.getFullYear();
	year.id = "chart-share-controls-year";
	chartControls.append(year);


	const resGithub = await fetch("https://api.github.com/user/" + res["github_user_id"]);
	if (result.status === 200) {
		const githubRes = await resGithub.json();

		const avatarIMG = document.createElement("img");
		avatarIMG.src = githubRes["avatar_url"];

		const githubAnchor = document.createElement("a");
		githubAnchor.href = githubRes["html_url"];
		githubAnchor.innerText = githubRes["login"];

		const createdBy = document.createElement("div");
		createdBy.id = "share-created-by";
		createdBy.append(avatarIMG)
		createdBy.append(githubAnchor)

		const wrapper = document.createElement("div");
		wrapper.id = "share-created-by-wrapper";
		wrapper.append("Chart created by: ");
		wrapper.append(createdBy);

		chartControls.append(wrapper);
	}

	const editButton = document.createElement("a");
	editButton.href = "/?forceedit&s=" + res["chart"];
	editButton.innerText = "Edit";
	editButton.classList.add("button", "button-yellow");
	chartControls.append(editButton);

	const gitReproducer = document.createElement("div");
	gitReproducer.id = "share-git-reproducer";
	gitReproducer.classList.add("flex-grow", "flex-column");

	const h2 = document.createElement("h2");
	h2.innerText = "Git cmd reproducer";
	gitReproducer.append(h2);

	const gitReproControls = document.createElement("div");
	gitReproControls.classList.add("flex-row", "flex-center", "gap-05");


	const commitMessageLabel = document.createElement("label");
	commitMessageLabel.classList.add("inputlabel");
	const commitMessageInput = document.createElement("input");
	commitMessageInput.classList.add("input");
	commitMessageInput.type = "text";
	commitMessageInput.autocomplete = "off";
	commitMessageInput.value = document.location.host + document.location.pathname;
	commitMessageLabel.append("Commit message: ", commitMessageInput);

	const code = document.createElement("code");
	code.id = "cmd";

	const commitMessageUpdate = () => {
		let cmds = "";
		chart.querySelectorAll(".clicked").forEach((node) => {
			if (cmds !== "") {
				cmds += "\n" + "git commit --date \"" + node.dataset.date + "\" -m \"" + commitMessageInput.value + "\""
			} else {
				cmds = "git commit --date \"" + node.dataset.date + "\" -m \"" + commitMessageInput.value + "\""
			}
		});
		code.innerText = cmds;
	};

	commitMessageInput.addEventListener("input", () => {
		commitMessageUpdate();
	});
	commitMessageUpdate();

	const copyButton = document.createElement("button");
	copyButton.addEventListener("click", () => {
		 navigator.clipboard.writeText(code.innerText);
	});
	copyButton.classList.add("button", "button-yellow");
	copyButton.innerText = "Copy to clipboard";

	gitReproControls.append(commitMessageLabel);
	gitReproControls.append(copyButton);

	gitReproducer.append(gitReproControls);

	const cmdWrapper = document.createElement("div");
	cmdWrapper.classList.add("cmd-wrapper");

	cmdWrapper.append(code);

	gitReproducer.append(cmdWrapper);

	document.getElementById("chart-share").append(chartControls, chart, gitReproducer);
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
