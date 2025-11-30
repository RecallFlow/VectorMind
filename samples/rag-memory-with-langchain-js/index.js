import { ChatOpenAI } from "@langchain/openai";
import prompts from "prompts";
//import dotenv from "dotenv";
//dotenv.config();

const vectormindApiUrl =
  process.env.VECTORMIND_API_URL || "http://localhost:8080";
// Define [CHAT MODEL] Connection
const chatModel = new ChatOpenAI({
  model: process.env.MODEL_RUNNER_LLM_CHAT || `ai/qwen2.5:latest`,
  apiKey: "",
  configuration: {
    baseURL: process.env.MODEL_RUNNER_BASE_URL,
  },
  temperature: parseFloat(process.env.OPTION_TEMPERATURE) || 0.0,
  top_p: parseFloat(process.env.OPTION_TOP_P) || 0.5,
});

// SYSTEM INSTRUCTIONS:
let systemInstructions =
  process.env.SYSTEM_INSTRUCTIONS ||
  `You are Bob, an advanced AI assistant developed to help users`;

let exit = false;
// CHAT LOOP:
while (!exit) {
  const { userQuestion } = await prompts({
    type: "text",
    name: "userQuestion",
    message: `🤖 Your question (${chatModel.model}): `,
    validate: (value) => (value ? true : "😡 Question cannot be empty"),
  });

  if (userQuestion == "/bye") {
    console.log("👋 See you later!");
    exit = true;
  }

  // Search similar document in memory
  const searchResult = await searchSimilar(userQuestion, "memory", 5, 1.0);
  const documents = searchResult.results.map((r) => r.content).join("\n");

  let messages = [
    ["system", systemInstructions],
    [
      "system",
      "Use the following documents to make your answer:\n" + documents,
    ],
    ["user", userQuestion],
  ];

  let answer = "";
  // STREAMING COMPLETION:
  const stream = await chatModel.stream(messages);
  for await (const chunk of stream) {
    process.stdout.write(chunk.content);
    answer += chunk.content;
  }

  // Save the interaction in memory
  const result = await createEmbedding(
    JSON.stringify({
      user: userQuestion,
      assistant: answer,
    }),
    "memory",
    "",
  );

  console.log("\n");
}

// Function to create embeddings
async function createEmbedding(content, label = "", metadata = "") {
  const response = await fetch(`${vectormindApiUrl}/embeddings`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      content,
      label,
      metadata,
    }),
  });

  return await response.json();
}

// Function to search for similar documents
async function searchSimilar(
  text,
  label = "",
  maxCount = 5,
  distanceThreshold = 0.7,
) {
  const response = await fetch(`${vectormindApiUrl}/search`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      text,
      label: label,
      max_count: maxCount,
      distance_threshold: distanceThreshold,
    }),
  });

  return await response.json();
}
