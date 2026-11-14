const button = document.getElementById("submitBtn");
button.addEventListener("click", submitForm);

function isValidEmail(value) {
  return value.includes("@");
}

function submitForm(event) {
  event.preventDefault();
  const email = document.querySelector("#email").value;
  if (!isValidEmail(email)) {
    document.getElementById("emailError").classList.add("visible");
    return;
  }
  fetch("/api/register", { method: "POST" });
}
