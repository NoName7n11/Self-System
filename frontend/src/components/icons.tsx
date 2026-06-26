// Pixel icon system — 7×7 grid SVGs, 2×2 rect per lit cell, currentColor fill.
// viewBox="0 0 14 14", shape-rendering:crispEdges. Cell coords [col,row] 0-indexed.

function px(cells: [number, number][]): React.ReactElement {
  return (
    <svg
      width={15}
      height={15}
      viewBox="0 0 14 14"
      fill="currentColor"
      style={{ display: "block", shapeRendering: "crispEdges" }}
    >
      {cells.map(([c, r], i) => (
        <rect key={i} x={c * 2} y={r * 2} width={2} height={2} />
      ))}
    </svg>
  );
}

export const IcoLogo = () =>
  px([[1,1],[1,2],[1,3],[1,4],[1,5],[5,1],[5,2],[5,3],[5,4],[5,5],[2,3],[3,3],[4,3]]);

export const IcoSearch = () =>
  px([[1,1],[2,1],[3,1],[1,2],[3,2],[1,3],[2,3],[3,3],[4,4],[5,5]]);

export const IcoChat = () =>
  px([[1,1],[2,1],[3,1],[4,1],[5,1],[1,2],[5,2],[1,3],[2,3],[3,3],[4,3],[5,3],[1,4]]);

export const IcoTasks = () =>
  px([[1,3],[2,4],[3,3],[4,2],[5,1]]);

export const IcoLibrary = () =>
  px([[1,1],[2,1],[3,1],[4,1],[5,1],[1,2],[5,2],[1,3],[2,3],[3,3],[4,3],[5,3],[1,4],[5,4],[1,5],[2,5],[3,5],[4,5],[5,5]]);

export const IcoGrid = () =>
  px([[1,1],[2,1],[1,2],[2,2],[4,1],[5,1],[4,2],[5,2],[1,4],[2,4],[1,5],[2,5],[4,4],[5,4],[4,5],[5,5]]);

export const IcoGear = () =>
  px([[3,1],[3,5],[1,3],[5,3],[2,2],[4,2],[2,4],[4,4],[3,3]]);

export const IcoPlus = () =>
  px([[3,1],[3,2],[3,3],[3,4],[3,5],[1,3],[2,3],[4,3],[5,3]]);

export const IcoSend = () =>
  px([[1,1],[1,2],[2,2],[1,3],[2,3],[3,3],[1,4],[2,4],[1,5]]);

export const IcoFilter = () =>
  px([[1,1],[2,1],[3,1],[4,1],[5,1],[2,3],[3,3],[4,3],[3,5]]);

export const IcoTrend = () =>
  px([[1,5],[2,4],[3,3],[4,2],[5,1],[3,1],[4,1],[5,2]]);

export const IcoChevL = () =>
  px([[4,1],[3,2],[2,3],[3,4],[4,5]]);

export const IcoChevR = () =>
  px([[2,1],[3,2],[4,3],[3,4],[2,5]]);

export const IcoChevUp = () =>
  px([[1,4],[2,3],[3,2],[4,3],[5,4]]);

export const IcoChevDn = () =>
  px([[1,2],[2,3],[3,4],[4,3],[5,2]]);
