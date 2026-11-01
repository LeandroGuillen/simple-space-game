import { cleanup, render } from "@solidjs/testing-library";
import { afterEach, describe, expect, it } from "vitest";
import { formatTransferLoss, TransferLossNotices } from "../../components/TransferLossNotices";
import type { TransferLossNotice } from "../types/domain";

afterEach(cleanup);

const notice: TransferLossNotice = {
  owner: 1, turn: 3500.1, transferId: 3, sourcePlanetId: 10, sourcePlanetName: "Sol I",
  destinationPlanetId: 20, destinationPlanetName: "Vega II", populationLost: 1200,
};

describe("transfer loss notices", () => {
  it("names the planets and the number lost", () => {
    const view = render(() => <TransferLossNotices notices={[notice]} playerId={1} />);
    expect(view.getByRole("status").textContent).toContain("1,200 colonists were lost in transit from Sol I to Vega II.");
  });

  it("uses planet IDs for missing names and hides another player's notices", () => {
    expect(formatTransferLoss({ ...notice, sourcePlanetName: "", destinationPlanetName: "" })).toContain("Planet #10 to Planet #20");
    const view = render(() => <TransferLossNotices notices={[notice]} playerId={2} />);
    expect(view.queryByRole("status")).toBeNull();
  });
});
