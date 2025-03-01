// Function to handle like/dislike for a post
async function reactToPost(userId, postId, likeType) {
  console.log('UserID:', userId);
    console.log('PostID:', postId);
    console.log('LikeType:', likeType);
  if (userId == 0) {
    window.location.href = "/login"
    return
  }
  
  try {
    const response = await fetch("/like", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        user_id: userId,
        post_id: postId,
        like_type: likeType,
      }),
    });

    const result = await response.json();

    if (response.ok) {
      updateReactionUI(
        postId,
        null,
        result.likes,
        result.dislikes,
        result.userReaction
      ); // Update UI
    }
  } catch (error) {
    console.error("Error:", error);
  }
}

// Function to handle like/dislike for a comment
async function reactToComment(userId, commentId, likeType) {
  if (userId == 0) {
    window.location.href = "/login"
    return
  }

  try {
    const response = await fetch("/like", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        user_id: userId,
        comment_id: commentId,
        like_type: likeType,
      }),
    });

    const result = await response.json();

    if (response.ok) {
      updateReactionUI(
        null,
        commentId,
        result.likes,
        result.dislikes,
        result.userReaction
      ); // Update UI
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
    dislikeCount = document.querySelector(
      `#comment-dislike-count-${commentId}`
    );
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

document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll(".toggle-comments").forEach((button) => {
    button.addEventListener("click", function () {
      let postId = this.getAttribute("data-post-id");
      let commentSection = document.getElementById(`comments-${postId}`);
      commentSection.classList.toggle("hidden");
    });
  });
});

function OpenComments(id) {
  let comments = document.getElementById(id);

  comments.classList.toggle("close");
}

document.addEventListener("DOMContentLoaded", function() {
  const urlParams = new URLSearchParams(window.location.search);
  if (urlParams.has('login_success')) {
    window.history.replaceState({}, document.title, window.Location.pathname);

      window.location.reload(true);
    }
  }
});