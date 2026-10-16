"use strict";

function requestPollResults(poll_element, pollId) {
    const post = poll_element.closest(".post");
    if (!post) return Promise.reject(new Error("poll block outside a post"));

    const blogNameElement = post.getElementsByClassName("blog-name")[0];
    const blogName = blogNameElement ? blogNameElement.textContent : "";
    const postId = post.dataset.postId;
    if (!blogName || !postId || !pollId) {
        return Promise.reject(new Error("incomplete poll reference"));
    }

    let poll_results_url = `/api/v1/poll/${encodeURIComponent(blogName)}/${encodeURIComponent(postId)}/${encodeURIComponent(pollId)}/results`;
    if (poll_element.classList.contains("expired-poll")) {
        poll_results_url += "?expired=1";
    }

    return fetch(poll_results_url).then((response) => {
        if (!response.ok) {
            throw new Error(`poll results request failed: HTTP ${response.status}`);
        }
        return response.json();
    });
}

function fill_poll_results(poll_element, results) {
    const pollChoices = poll_element.getElementsByClassName("poll-choices")[0];
    const pollMetadataElement = poll_element.getElementsByClassName("poll-metadata")[0];
    if (!pollChoices || !pollMetadataElement) return;
    const pollTimestamp = pollMetadataElement.children[0];
    if (!pollTimestamp) return;

    const votes_by_answer = (results && results.results) || {};
    const sorted_poll_results = Object.entries(votes_by_answer).sort((a,b) => (a[1]-b[1])).reverse();

    // First we must find the total number of votes and the winner(s) of the poll
    let total_votes = 0;

    // Answer ID to winner status and amount of votes
    const processed_poll_results = {};
    const winner_vote_count = sorted_poll_results.length ? sorted_poll_results[0][1] : 0;

    for (let [answer_id, votes] of sorted_poll_results) {
        processed_poll_results[answer_id] = {"is_winner": (winner_vote_count == votes), "votes": votes};
        total_votes += votes
    }

    const answerIdChoiceElementArray = [];

    for (let choiceElement of pollChoices.children) {
        answerIdChoiceElementArray.push([choiceElement.dataset.answerId, choiceElement]);
    }

    for (let [answer_id, choiceElement] of answerIdChoiceElementArray) {
        const answer_results = processed_poll_results[answer_id] || {"is_winner": false, "votes": 0};
        const is_winner = answer_results.is_winner;
        const answer_votes = answer_results.votes;

        let numericalVoteProportion

        if (answer_votes == 0 || total_votes == 0) {
            numericalVoteProportion = 0;
        } else {
            numericalVoteProportion = answer_votes/total_votes;
        }

        const voteProportionElement = document.createElement("span");
        voteProportionElement.classList.add("vote-proportion");
        voteProportionElement["style"] = `width: ${((numericalVoteProportion) * 100).toFixed(3)}%;`;

        const voteCountElement = document.createElement("p");
        voteCountElement.classList.add("vote-count");

        // A greater rounding precision is needed here
        const comparison = Math.round((numericalVoteProportion) * 10000)/10000
        if ((comparison > 0.001) || comparison == 0) {
            voteCountElement.innerHTML = new Intl.NumberFormat("en-US", {style: "percent", maximumSignificantDigits: 3}).format(Math.round((numericalVoteProportion) * 1000)/1000);
        } else {
            voteCountElement.innerHTML = "< 0.1%";
        }

        if (is_winner) {
            choiceElement.classList.add("poll-winner");
        }

        choiceElement.appendChild(voteProportionElement);
        choiceElement.appendChild(voteCountElement);
    }

    const totalVotesElement = document.createElement("span")
    totalVotesElement.textContent = `${total_votes} votes`

    const separatorElement = document.createElement("span")
    separatorElement.textContent = '•'
    separatorElement.classList.add("separator")

    const newMetadataContents = document.createDocumentFragment()
    newMetadataContents.append(totalVotesElement, separatorElement, pollTimestamp.cloneNode(true))

    pollTimestamp.replaceWith(newMetadataContents)

    poll_element.classList.add("populated")
}

const pollBlocks = document.getElementsByClassName("poll-block");


function populate_polls() {
    for (let poll of pollBlocks) {
        if (poll.classList.contains("populated")) {
            continue;
        }

        requestPollResults(poll, poll.dataset.pollId)
            .then((answers) => { fill_poll_results(poll, answers) })
            .catch(() => { /* leave the poll as rendered; results are optional */ });
    }
};

// TODO lazy load polls
populate_polls();

// Control-bar dropdowns open on hover (CSS) and on click/tap (here), so they
// work with touch and keyboard as well as a mouse.
(function () {
    const actions = document.querySelectorAll(".control-bar-action");
    if (!actions.length) return;

    function closeAll(except) {
        actions.forEach(function (action) {
            if (action !== except) action.classList.remove("open");
        });
    }

    actions.forEach(function (action) {
        const menu = action.querySelector(".control-bar-dropdown-menu");
        if (menu) {
            action.setAttribute("aria-haspopup", "true");
            action.setAttribute("aria-expanded", "false");
        }
        action.addEventListener("click", function (event) {
            if (event.target.closest("a")) return; // menu link: navigate
            event.preventDefault();
            const willOpen = !action.classList.contains("open");
            closeAll(action);
            action.classList.toggle("open", willOpen);
            action.setAttribute("aria-expanded", willOpen ? "true" : "false");
        });
    });

    document.addEventListener("click", function (event) {
        if (event.target.closest && event.target.closest(".control-bar-action")) return;
        closeAll(null);
    });
    document.addEventListener("keydown", function (event) {
        if (event.key === "Escape") closeAll(null);
    });
})();

// Offline shell + fast repeat loads (progressive web app).
if ('serviceWorker' in navigator) {
    window.addEventListener('load', function () {
        navigator.serviceWorker.register('/sw.js', { scope: '/' }).catch(function () {});
    });
}
