interface GreetingResponse {
  message: string;
}

const greetingNode = document.getElementById("greeting");

async function loadGreeting(): Promise<void> {
  if (!greetingNode) {
    return;
  }

  try {
    const response = await fetch("/api/greeting");
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const data = (await response.json()) as GreetingResponse;
    greetingNode.textContent = data.message;
  } catch (error) {
    console.error(error);
    greetingNode.textContent = "Не удалось загрузить приветствие. Попробуйте позже.";
  }
}

void loadGreeting();
