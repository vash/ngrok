function toggleVisibility(key) {
  let hiddenField = document.getElementById(`key-${key}`);
  let placeholder = document.getElementById(`key-placeholder-${key}`);

  if (hiddenField && placeholder) {
    hiddenField.classList.toggle("hidden");
    placeholder.classList.toggle("hidden");
  }
}

function copyToClipboard(key) {
  let hiddenField = document.getElementById(`key-${key}`);
  let copyButton = document.getElementById(`copy-btn-${key}`);

  if (hiddenField && copyButton) {
    navigator.clipboard.writeText(hiddenField.innerText).then(() => {
      copyButton.innerHTML = "V";

      // Revert back after 2 seconds
      setTimeout(() => {
        copyButton.innerHTML = "C";
      }, 2000);
    });
  }
}
