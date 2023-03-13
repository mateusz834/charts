document.addEventListener("DOMContentLoaded", async () => {
	const loggedAS = document.getElementById("logged-as");
	const githubProfileAnchor = document.getElementById("github-profile-anchor");
	const githubProfileAvatar = document.getElementById("github-profile-avatar");
	const loginWithGithub = document.getElementById("login-with-github");
	const moreOptions = document.getElementById("more-options");
	const moreOptionsSection = document.getElementById("more-options-section");
	const result = await fetch("/user-info", { method: "POST" });

	if (result.status === 200) {
		const res = await result.json();
		if (res["github_user_id"] !== undefined) {
			window.loggedUser = {
				githubUserID: res["github_user_id"],
				githubLogin: null,
				githubProfileURL: null,
			};

			const result = await fetch("https://api.github.com/user/" + res["github_user_id"]);
			if (result.status === 200) {
				const githubRes = await result.json();
				githubProfileAvatar.src = githubRes["avatar_url"];
				githubProfileAnchor.href = githubRes["html_url"];
				githubProfileAnchor.innerText = githubRes["login"];
				loginWithGithub.classList.add("hidden");
				loggedAS.classList.remove("hidden");
				window.loggedUser.githubLogin = githubRes["login"];
				window.loggedUser.githubProfileURL = githubRes["html_url"];
			}
		}
	}

	moreOptions.addEventListener("click", () => {
		moreOptionsSection.classList.toggle("hidden");
	});

	document.addEventListener("click", (e) => {
		if (!moreOptionsSection.classList.contains("hidden")) {
			if (e.target !== moreOptions && e.target !== moreOptionsSection) {
				moreOptionsSection.classList.add("hidden")
			}
		}
	});
});
