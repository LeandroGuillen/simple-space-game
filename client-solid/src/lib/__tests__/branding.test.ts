import { readFileSync } from "node:fs";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

import { planetArtSlugByEnvironment } from "../art";
import {
  APP_NAME,
  formatDocumentTitle,
  screenTitleForPath,
} from "../document-title";

const clientRoot = join(process.cwd());
const publicDir = join(clientRoot, "public");
const indexHtml = readFileSync(join(clientRoot, "index.html"), "utf-8");

describe("SSG branding", () => {
  it("uses SSG as the static document title in index.html", () => {
    expect(indexHtml).toMatch(/<title>\s*SSG\s*<\/title>/);
    expect(indexHtml).not.toMatch(/client-solid/i);
  });

  it("links favicon, touch icon, and web manifest in index.html", () => {
    expect(indexHtml).toContain('href="/favicon.svg"');
    expect(indexHtml).toContain('href="/favicon-32x32.png"');
    expect(indexHtml).toContain('href="/favicon-16x16.png"');
    expect(indexHtml).toContain('href="/apple-touch-icon.png"');
    expect(indexHtml).toContain('href="/site.webmanifest"');
    expect(indexHtml).toContain('content="#06101a"');
  });

  it("ships a non-template favicon and manifest assets", () => {
    const faviconSvg = readFileSync(join(publicDir, "favicon.svg"), "utf-8");
    expect(faviconSvg).not.toContain("863bff");
    expect(faviconSvg).toContain("#80c6c3");

    const manifest = JSON.parse(
      readFileSync(join(publicDir, "site.webmanifest"), "utf-8"),
    ) as { name: string; short_name: string; theme_color: string; background_color: string };
    expect(manifest.name).toBe("SSG");
    expect(manifest.short_name).toBe("SSG");
    expect(manifest.theme_color).toBe("#06101a");
    expect(manifest.background_color).toBe("#06101a");

    for (const file of [
      "favicon-16x16.png",
      "favicon-32x32.png",
      "apple-touch-icon.png",
      "android-chrome-192x192.png",
      "android-chrome-512x512.png",
    ]) {
      expect(existsSync(join(publicDir, file)), file).toBe(true);
    }
  });

  it("resolves every planet environment slug to an on-disk art file", () => {
    for (const slug of Object.values(planetArtSlugByEnvironment)) {
      const numbered = join(publicDir, "art", "planets", `${slug}1.png`);
      const unnumbered = join(publicDir, "art", "planets", `${slug}.png`);
      expect(
        existsSync(numbered) || existsSync(unnumbered),
        `missing art for slug ${slug}`,
      ).toBe(true);
    }
  });

  it("formats in-game document titles with game name and turn", () => {
    expect(
      formatDocumentTitle({
        pathname: "/galaxy",
        hasGame: true,
        playerId: 2,
        game: {
          id: "abc",
          currentTurn: 7,
          players: [
            { id: 1, name: "Alpha" },
            { id: 2, name: "Beta Empire" },
          ],
        },
      }),
    ).toBe("SSG - Beta Empire - Turn 7");
  });

  it("formats lobby and setup titles with screen names", () => {
    expect(
      formatDocumentTitle({
        pathname: "/race-editor",
        hasGame: false,
      }),
    ).toBe("SSG - Race Editor");
    expect(screenTitleForPath("/")).toBe("Setup");
    expect(APP_NAME).toBe("SSG");
  });
});
