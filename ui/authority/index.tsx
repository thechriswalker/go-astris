import { createElement } from "react";
import { render } from "react-dom";
import { Election } from "./Election";

// and render
render(createElement(Election), document.getElementById("app"));
