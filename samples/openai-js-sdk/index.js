import OpenAI from "openai";

// OpenAI Client
const openai = new OpenAI({
	baseURL: "http://localhost:12434/engines/v1",
	apiKey: "i-love-docker-model-runner",
});

const VECTORMIND_API = "http://localhost:8080";

const chunks = [
	`# Orcs
	Orcs are savage, brutish humanoids with dark green skin and prominent tusks. 
	These fierce warriors inhabit dense forests where they hunt in packs, 
	using crude but effective weapons forged from scavenged metal and bone. 
	Their tribal society revolves around strength and combat prowess, 
	making them formidable opponents for any adventurer brave enough to enter their woodland domain.`,

	`# Dragons
	Dragons are magnificent and ancient creatures of immense power, soaring through the skies on massive wings. 
	These intelligent beings possess scales that shimmer like precious metals and breathe devastating elemental attacks. 
	Known for their vast hoards of treasure and centuries of accumulated knowledge, 
	dragons command both fear and respect throughout the realm. 
	Their aerial dominance makes them nearly untouchable in their celestial domain.`,

	`# Goblins
	Goblins are small, cunning creatures with mottled green skin and sharp, pointed ears. 
	Despite their diminutive size, they are surprisingly agile swimmers who have adapted to life around ponds and marshlands. 
	These mischievous beings are known for their quick wit and tendency to play pranks on unwary travelers. 
	They build elaborate underwater lairs connected by hidden tunnels beneath the murky pond waters.`,

	`# Krakens
	Krakens are colossal sea monsters with massive tentacles that can crush entire ships with ease. 
	These legendary creatures dwell in the deepest ocean trenches, surfacing only to hunt or when disturbed. 
	Their intelligence rivals that of the wisest sages, and their tentacles can stretch for hundreds of feet. 
	Sailors speak in hushed tones of these maritime titans, whose very presence can create devastating whirlpools 
	and tidal waves that reshape entire coastlines.`,
];


// Function to search for similar documents
async function searchSimilar(text, maxCount = 5, distanceThreshold = 0.7) {
	const response = await fetch(`${VECTORMIND_API}/search`, {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify({
			text,
			max_count: maxCount,
			distance_threshold: distanceThreshold,
		}),
	});

	return await response.json();
}

let userInput = "Tell me something about the dragons";

try {
	// Create embeddings from chunks
	console.log("Creating embeddings...\n");

	for (const chunk of chunks) {
		const result = await createEmbedding(chunk, "fantasy-creatures", "");
		console.log("Created embedding:", result);
	}

	// Search for similar documents
	console.log("\n\nSearching for similar documents...\n");

	const searchResult = await searchSimilar(userInput, 1, 0.7);
	console.log("Search results:\n", JSON.stringify(searchResult, null, 2));

	const documents = searchResult.results.map(r => r.content).join("\n");


	const completion = await openai.chat.completions.create({
		model: "hf.co/menlo/jan-nano-gguf:q4_k_m",
		messages: [
      { role: "system", content: "Using the following documents:" },
      { role: "system", content: "documents:\n"+ documents },
      { role: "user", content: "userInput" }
    ],
		stream: true,
	});

  console.log("=".repeat);

	for await (const chunk of completion) {
		process.stdout.write(chunk.choices[0].delta.content || "");
	}
} catch (error) {
	console.error("Error:", error);
}
