import * as React from "react";
import qrcode = require("qr.js");

// 4px squares.
const PIXEL_SIZE = 4;
// assume we are at the topleft
const SQUARE = `v${PIXEL_SIZE}h${PIXEL_SIZE}v-${PIXEL_SIZE}z`;

// Create a QR code as an SVG.
export const QR: React.FC<{ data: string }> = ({ data }) => {
  const cells = qrcode(data, {
    errorCorrectLevel: qrcode.ErrorCorrectLevel.M,
  }).modules;
  // now qr.modules is an array of rows, each with an array of cells, either true or false
  const paths: string[] = [];
  const rows = cells.length;
  for (let r = 0; r < rows; r++) {
    const row = cells[r];
    // each row we move down a pixel. This gives a single pixel padding.
    // maybe we should move 2 the first time.
    // but we do it absolutely so we reset the left margin each row
    // irrespective of where we got to.
    paths.push(`M${1 * PIXEL_SIZE},${PIXEL_SIZE * (r + 2)}`);
    let m = 1;
    for (let c = 0; c < row.length; c++) {
      // if the cell is "ON" draw a square
      if (row[c]) {
        paths.push(`m${m * PIXEL_SIZE},0`, SQUARE);
        m = 1; // 1 will bring us back to the top right of this square
      } else {
        m++; // increase the amount we have to move when we next draw a square
      }
    }
  }
  const [width, height] = [
    PIXEL_SIZE * (cells[0].length + 4),
    PIXEL_SIZE * (rows + 4),
  ];

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      aria-hidden="true"
      focusable="false"
      role="img"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path fill="currentColor" d={paths.join("")} />
    </svg>
  );
};
