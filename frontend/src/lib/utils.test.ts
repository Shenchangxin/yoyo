import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { cn } from "./utils.ts";

describe("cn", () => {
  it("keeps a text-ui size next to a text color", () => {
    assert.equal(cn("text-ui-caption", "text-background"), "text-ui-caption text-background");
    assert.equal(cn("block text-ui-footnote text-muted"), "block text-ui-footnote text-muted");
  });

  it("lets a later text-ui size replace an earlier one", () => {
    assert.equal(cn("text-ui-body font-medium", "text-ui-caption"), "font-medium text-ui-caption");
  });
});
