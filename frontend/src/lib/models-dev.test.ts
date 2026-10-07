import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { hydrateCatalog, overlayCatalog, type ModelsDevCatalog } from "./catalog-shape.ts";
import { composerModelGroups, encodeChatModel, resolveComposerModelValue } from "./models-dev.ts";

const base: ModelsDevCatalog = {
  version: 3,
  fetchedAt: "2026-08-03T00:00:00Z",
  source: "snapshot",
  providers: {
    openai: {
      "gpt-4o": { model: { id: "gpt-4o", name: "GPT-4o", contextWindow: 128000 }, family: "gpt", releaseDate: "2024-05-13" },
    },
    grok: {
      "grok-4": { model: { id: "grok-4", name: "Grok 4", contextWindow: 256000 }, family: "grok" },
    },
  },
};

describe("overlayCatalog", () => {
  it("keeps bundled providers when live is empty", () => {
    const got = overlayCatalog(base, { version: 3, fetchedAt: "", providers: {} });
    assert.equal(got.providers.openai["gpt-4o"].model.id, "gpt-4o");
    assert.equal(got.providers.grok["grok-4"].model.id, "grok-4");
  });

  it("replaces a provider and keeps the rest", () => {
    const live: ModelsDevCatalog = {
      version: 3,
      fetchedAt: "2026-10-06T00:00:00Z",
      source: "models.dev",
      providers: {
        openai: {
          "gpt-5.4": { model: { id: "gpt-5.4", name: "GPT-5.4", contextWindow: 1050000 }, family: "gpt", releaseDate: "2026-03-05" },
        },
      },
    };
    const got = overlayCatalog(base, live);
    assert.equal(got.source, "models.dev");
    assert.equal(got.providers.openai["gpt-5.4"].model.contextWindow, 1050000);
    assert.equal(got.providers.openai["gpt-4o"], undefined);
    assert.equal(got.providers.grok["grok-4"].model.id, "grok-4");
  });
});

describe("hydrateCatalog", () => {
  it("reads Wails PascalCase wrappers", () => {
    const got = hydrateCatalog({
      Version: 3,
      FetchedAt: "2026-10-06T12:00:00Z",
      Source: "models.dev",
      Providers: {
        claude: {
          "claude-sonnet-4-6": {
            Family: "claude-sonnet",
            ReleaseDate: "2026-02-17",
            Model: { ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", ContextWindow: 1000000, Reasoning: true, Input: ["text", "image"] },
          },
        },
      },
    }, base);
    assert.equal(got.providers.claude["claude-sonnet-4-6"].model.contextWindow, 1_000_000);
    assert.equal(got.providers.grok["grok-4"].model.id, "grok-4");
  });
});

describe("composerModelGroups", () => {
  const providers = [
    { id: "a", name: "OpenAI", vendor: "openai", endpoint: "https://api.openai.com/v1", model: "gpt-4.1-mini", models: ["gpt-4.1-mini", "gpt-4.1"], hasKey: true, isDefault: true, active: true },
    { id: "b", name: "FlowY", vendor: "custom", endpoint: "https://server.flowyaipc.com/claw/v1", model: "openclaw/default", models: ["openclaw/default"], hasKey: true, isDefault: false, active: true },
  ];

  it("groups models under each provider", () => {
    const groups = composerModelGroups(providers, "gpt-4.1-mini", "a");
    assert.equal(groups.length, 2);
    assert.equal(groups[0].providerName, "OpenAI");
    assert.deepEqual(groups[0].models, ["gpt-4.1-mini", "gpt-4.1"]);
    assert.equal(groups[1].providerName, "FlowY");
    assert.deepEqual(groups[1].models, ["openclaw/default"]);
  });

  it("encodes provider and model so two catalogs can share an id", () => {
    assert.equal(encodeChatModel("a", "gpt-4.1-mini"), "a::gpt-4.1-mini");
    assert.equal(resolveComposerModelValue(providers, "openclaw/default", "b"), "b::openclaw/default");
    assert.equal(resolveComposerModelValue(providers, "gpt-4.1-mini"), "a::gpt-4.1-mini");
  });
});
