// main.js

// Wait for the DOM to load
document.addEventListener("DOMContentLoaded", function () {
    // Add event listeners and initialize functionality
    initFormValidation();
    initLikeDislikeButtons();
    initFilterPosts();
    initDynamicContentLoading();
});

// Form Validation
function initFormValidation() {
    const forms = document.querySelectorAll("form");

    forms.forEach((form) => {
        form.addEventListener("submit", function (event) {
            let isValid = true;

            // Validate required fields
            const inputs = form.querySelectorAll("input[required], textarea[required], select[required]");
            inputs.forEach((input) => {
                if (!input.value.trim()) {
                    isValid = false;
                    input.classList.add("error");
                } else {
                    input.classList.remove("error");
                }
            });

            if (!isValid) {
                event.preventDefault();
                alert("Please fill out all required fields.");
            }
        });
    });
}

// Like/Dislike Buttons
function initLikeDislikeButtons() {
    const likeButtons = document.querySelectorAll(".like-button");
    const dislikeButtons = document.querySelectorAll(".dislike-button");

    likeButtons.forEach((button) => {
        button.addEventListener("click", function () {
            const postId = button.dataset.postId;
            handleLikeDislike(postId, "like");
        });
    });

    dislikeButtons.forEach((button) => {
        button.addEventListener("click", function () {
            const postId = button.dataset.postId;
            handleLikeDislike(postId, "dislike");
        });
    });
}

// Handle Like/Dislike
function handleLikeDislike(postId, action) {
    fetch(`/post/${postId}/${action}`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
    })
        .then((response) => response.json())
        .then((data) => {
            if (data.success) {
                // Update the like/dislike count on the page
                const likeCountElement = document.querySelector(`#like-count-${postId}`);
                const dislikeCountElement = document.querySelector(`#dislike-count-${postId}`);

                if (likeCountElement && dislikeCountElement) {
                    likeCountElement.textContent = data.likeCount;
                    dislikeCountElement.textContent = data.dislikeCount;
                }
            } else {
                alert("Failed to update like/dislike. Please try again.");
            }
        })
        .catch((error) => {
            console.error("Error:", error);
        });
}

// Filter Posts
function initFilterPosts() {
    const filterForm = document.querySelector("#filter-form");

    if (filterForm) {
        filterForm.addEventListener("submit", function (event) {
            event.preventDefault();

            const category = document.querySelector("#filter-category").value;
            const sortBy = document.querySelector("#filter-sort").value;

            // Fetch filtered posts
            fetch(`/posts/filter?category=${category}&sort=${sortBy}`)
                .then((response) => response.json())
                .then((data) => {
                    if (data.success) {
                        // Update the posts section with the filtered posts
                        const postsSection = document.querySelector("#posts-section");
                        postsSection.innerHTML = ""; // Clear existing posts

                        data.posts.forEach((post) => {
                            const postElement = document.createElement("div");
                            postElement.classList.add("post");
                            postElement.innerHTML = `
                                <h2>${post.title}</h2>
                                <p><strong>Posted by:</strong> ${post.username} | <strong>Category:</strong> ${post.category}</p>
                                <p>${post.content}</p>
                                <p><strong>Likes:</strong> ${post.likeCount} | <strong>Dislikes:</strong> ${post.dislikeCount}</p>
                            `;
                            postsSection.appendChild(postElement);
                        });
                    } else {
                        alert("Failed to filter posts. Please try again.");
                    }
                })
                .catch((error) => {
                    console.error("Error:", error);
                });
        });
    }
}

// Dynamic Content Loading (e.g., comments)
function initDynamicContentLoading() {
    const loadMoreButtons = document.querySelectorAll(".load-more");

    loadMoreButtons.forEach((button) => {
        button.addEventListener("click", function () {
            const postId = button.dataset.postId;
            const offset = button.dataset.offset;

            // Fetch more comments
            fetch(`/post/${postId}/comments?offset=${offset}`)
                .then((response) => response.json())
                .then((data) => {
                    if (data.success) {
                        // Append new comments to the comments section
                        const commentsSection = document.querySelector(`#comments-section-${postId}`);
                        data.comments.forEach((comment) => {
                            const commentElement = document.createElement("div");
                            commentElement.classList.add("comment");
                            commentElement.innerHTML = `
                                <p><strong>${comment.username}</strong> (${comment.createdAt}):</p>
                                <p>${comment.content}</p>
                            `;
                            commentsSection.appendChild(commentElement);
                        });

                        // Update the offset for the next load
                        button.dataset.offset = parseInt(offset) + data.comments.length;

                        // Hide the button if there are no more comments to load
                        if (data.comments.length < 5) {
                            button.style.display = "none";
                        }
                    } else {
                        alert("Failed to load more comments. Please try again.");
                    }
                })
                .catch((error) => {
                    console.error("Error:", error);
                });
        });
    });
}

// Function to handle like/dislike for a post
async function reactToPost(userId, postId, likeType) {
    try {
        const response = await fetch("/like", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ 
                user_id: userId,
                post_id: postId,
                like_type: likeType
            })
        });

        const result = await response.json();
        console.log(result.message);

        if (response.ok) {
            updateReactionUI(postId, null, result.likes, result.dislikes, result.userReaction);  // Update UI
        }
    } catch (error) {
        console.error("Error:", error);
    }
}

// Function to handle like/dislike for a comment
async function reactToComment(userId, commentId, likeType) {
    try {
        const response = await fetch("/like", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ 
                user_id: userId,
                comment_id: commentId,
                like_type: likeType
            })
        });

        const result = await response.json();
        console.log(result.message);

        if (response.ok) {
            updateReactionUI(null, commentId, result.likes, result.dislikes, result.userReaction);  // Update UI
        }
    } catch (error) {
        console.error("Error:", error);
    }
}

// Function to update the UI instantly without reloading
function updateReactionUI(postId, commentId, likes, dislikes, userReaction) {
    let likeBtn, dislikeBtn, likeCount, dislikeCount;

    if (postId) {
        likeBtn = document.querySelector(`#like-post-${postId}`);
        dislikeBtn = document.querySelector(`#dislike-post-${postId}`);
        likeCount = document.querySelector(`#post-like-count-${postId}`);
        dislikeCount = document.querySelector(`#post-dislike-count-${postId}`);
    } else if (commentId) {
        likeBtn = document.querySelector(`#like-comment-${commentId}`);
        dislikeBtn = document.querySelector(`#dislike-comment-${commentId}`);
        likeCount = document.querySelector(`#comment-like-count-${commentId}`);
        dislikeCount = document.querySelector(`#comment-dislike-count-${commentId}`);
    }

    if (!likeBtn || !dislikeBtn || !likeCount || !dislikeCount) {
        console.error("UI elements not found");
        return;
    }

    // Update counts with real backend data
    likeCount.innerText = likes;
    dislikeCount.innerText = dislikes;

    // Reset button states
    likeBtn.classList.remove("active");
    dislikeBtn.classList.remove("active");

    if (userReaction === "like") {
        likeBtn.classList.add("active");
    } else if (userReaction === "dislike") {
        dislikeBtn.classList.add("active");
    }
}
