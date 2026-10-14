import ts from "typescript";
import { describe, expect, it } from "vitest";

import routePermissions from "@nexul/client-core/routePermissions.json";

type Routes = Record<string, readonly string[]>;

const WRITE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);
const SPAN = "\u0000";

// Source text of every web and phone module that can send a request, tests excluded; read from disk, since a Vite import loads the phone tsconfig that needs its packages; vitest runs from web/.
const root = `${ts.sys.resolvePath(`${ts.sys.getCurrentDirectory()}/..`)}/`;
const modules: Record<string, string> = Object.fromEntries(
  ts.sys
    .readDirectory(root, [".ts", ".tsx"], ["**/*.test.ts", "**/*.test.tsx", "**/*.d.ts", "web/src/test"], ["web/src/**/*", "native/src/**/*"])
    .map((file) => [`../${file.slice(root.length)}`, ts.sys.readFile(file) ?? ""]),
);

// What a URL expression can resolve through: the file's own constants and the arrow helpers every module defines.
interface Reach {
  constants: Map<string, ts.Expression>;
  helpers: Map<string, ts.Expression>;
}

// Every pattern a URL expression can send to, its substituted parts SPAN, or null when the call doesn't show them.
const patternsOf = (node: ts.Expression, reach: Reach, depth = 0): string[] | null => {
  if (depth > 5) return null;
  if (ts.isStringLiteralLike(node)) return [node.text];
  if (ts.isParenthesizedExpression(node) || ts.isAsExpression(node)) return patternsOf(node.expression, reach, depth + 1);
  if (ts.isConditionalExpression(node)) {
    const [yes, no] = [patternsOf(node.whenTrue, reach, depth + 1), patternsOf(node.whenFalse, reach, depth + 1)];
    return yes && no ? [...yes, ...no] : null;
  }
  if (ts.isIdentifier(node)) {
    const init = reach.constants.get(node.text);
    return init ? patternsOf(init, reach, depth + 1) : null;
  }
  if (ts.isCallExpression(node) && ts.isIdentifier(node.expression)) {
    const body = reach.helpers.get(node.expression.text);
    return body ? patternsOf(body, reach, depth + 1) : null;
  }
  if (!ts.isTemplateExpression(node)) return null;
  return node.templateSpans.reduce(
    (prefixes, span) => {
      const inner = (!ts.isIdentifier(span.expression) && patternsOf(span.expression, reach, depth + 1)) || [SPAN];
      return prefixes.flatMap((prefix) => inner.map((part) => prefix + part + span.literal.text));
    },
    [node.head.text],
  );
};

const parse = (file: string, text: string) =>
  ts.createSourceFile(file, text, ts.ScriptTarget.ES2022, true, file.endsWith("x") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);

// The arrow functions a module defines at its top level, by name, as the expression they return.
const helpersOf = (source: ts.SourceFile): [string, ts.Expression][] =>
  source.statements.filter(ts.isVariableStatement).flatMap((statement) =>
    statement.declarationList.declarations.flatMap((decl) => {
      const fn = decl.initializer;
      if (!ts.isIdentifier(decl.name) || !fn || !ts.isArrowFunction(fn)) return [];
      const ret = ts.isBlock(fn.body) ? fn.body.statements.find(ts.isReturnStatement)?.expression : fn.body;
      return ret ? [[decl.name.text, ret] as [string, ts.Expression]] : [];
    }),
  );

const constantsOf = (source: ts.SourceFile): Map<string, ts.Expression> => {
  const found = new Map<string, ts.Expression>();
  const walk = (node: ts.Node) => {
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.initializer && !ts.isArrowFunction(node.initializer) && !found.has(node.name.text)) {
      found.set(node.name.text, node.initializer);
    }
    ts.forEachChild(node, walk);
  };
  walk(source);
  return found;
};

const segments = (path: string) => path.replace(/\?.*$/, "").split("/");

// Routes of this method the URL can reach, keeping the most specific: a literal segment equal to a static one, or a substitution in a parameter's place, scores.
const matchRoutes = (method: string, url: string, routes: Routes): string[] => {
  const want = segments(url.startsWith(SPAN) && url[1] === "/" ? url.slice(1) : url);
  const scored: [string, number][] = [];
  for (const route of Object.keys(routes)) {
    const [routeMethod, routePath] = route.split(" ") as [string, string];
    const have = segments(routePath);
    if (routeMethod !== method || have.length !== want.length) continue;
    let score = 0;
    const fits = have.every((seg, i) => {
      const part = want[i] ?? "";
      if (seg.startsWith("{")) {
        if (part.includes(SPAN)) score++;
        return true;
      }
      if (part.includes(SPAN)) return true;
      if (part !== seg) return false;
      score++;
      return true;
    });
    if (fits) scored.push([route, score]);
  }
  const best = Math.max(0, ...scored.map(([, score]) => score));
  return scored.filter(([, score]) => score === best).map(([route]) => route);
};

interface Declaration {
  permission: string | null;
  line: number;
  writes: number;
}

// Every problem in one module: a request with no declaration, one the table contradicts, a declaration around no request.
const problemsIn = (file: string, source: ts.SourceFile, routes: Routes, helpers: Map<string, ts.Expression>): { problems: string[]; writes: number } => {
  const reach: Reach = { constants: constantsOf(source), helpers };
  const problems: string[] = [];
  const declarations: Declaration[] = [];
  let writes = 0;
  const at = (node: ts.Node) => `${file}:${source.getLineAndCharacterOfPosition(node.getStart()).line + 1}`;

  const writeOf = (call: ts.CallExpression): string | null => {
    const callee = call.expression;
    if (ts.isPropertyAccessExpression(callee) && ts.isIdentifier(callee.expression) && callee.expression.text === "api") {
      const method = callee.name.text.toUpperCase();
      return WRITE_METHODS.has(method) ? method : null;
    }
    if (!ts.isIdentifier(callee) || callee.text !== "requestJSON") return null;
    const options = call.arguments[1];
    if (!options || !ts.isObjectLiteralExpression(options)) return null;
    const method = options.properties.find((p): p is ts.PropertyAssignment => ts.isPropertyAssignment(p) && p.name.getText() === "method");
    const value = method && ts.isStringLiteralLike(method.initializer) ? method.initializer.text : "";
    return WRITE_METHODS.has(value) ? value : null;
  };

  const check = (call: ts.CallExpression, method: string, declared: Declaration) => {
    writes++;
    declared.writes++;
    const first = call.arguments[0];
    const urls = first ? patternsOf(first, reach) : null;
    if (urls === null) return void problems.push(`${at(call)} sends to a URL that can't be read; write it as a string or template literal`);
    if (declared.permission === null) return;
    for (const url of urls) {
      const path = url.startsWith(SPAN) && url[1] === "/" ? url.slice(1) : url;
      if (!path.startsWith("/api/")) {
        if (declared.permission !== "none") problems.push(`${at(call)} ${method} ${path} is outside the gateway and must declare "none"`);
        continue;
      }
      const matched = matchRoutes(method, url, routes);
      if (matched.length === 0) problems.push(`${at(call)} ${method} ${path} matches no route in routePermissions.json`);
      for (const route of matched) {
        if (!routes[route]?.includes(declared.permission)) problems.push(`${at(call)} declares "${declared.permission}" for ${route}, which takes ${JSON.stringify(routes[route])}`);
      }
    }
  };

  const visit = (node: ts.Node, declared: Declaration | undefined) => {
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === "requires") {
      const first = node.arguments[0];
      const next: Declaration = { permission: first && ts.isStringLiteralLike(first) ? first.text : null, line: 0, writes: 0 };
      if (next.permission === null) problems.push(`${at(node)} requires() takes its permission as a string literal`);
      declarations.push(next);
      node.arguments.forEach((arg) => visit(arg, next));
      if (next.writes === 0) problems.push(`${at(node)} requires() wraps no request`);
      return;
    }
    if (ts.isCallExpression(node)) {
      const method = writeOf(node);
      if (method && !declared) problems.push(`${at(node)} sends a ${method} and declares no permission; wrap it in requires()`);
      if (method && declared) check(node, method, declared);
      if (method && !declared) writes++;
    }
    ts.forEachChild(node, (child) => visit(child, declared));
  };
  visit(source, undefined);
  return { problems, writes };
};

describe("the writes the web and the phone send", () => {
  it("each declare the permission the server's route table names", () => {
    const sources = Object.entries(modules).map(([file, text]) => parse(file.replace("../", ""), text));
    const helpers = new Map(sources.flatMap(helpersOf));
    const all = sources.map((source) => problemsIn(source.fileName, source, routePermissions, helpers));
    expect(all.flatMap((r) => r.problems)).toEqual([]);
    // A scan that found nothing would pass the line above; 200 is far below today's count and far above a broken walk.
    expect(all.reduce((sum, r) => sum + r.writes, 0)).toBeGreaterThan(200);
  });
});

describe("the check", () => {
  const routes: Routes = {
    "PATCH /api/tickets/{id}/status": ["tickets:write"],
    "DELETE /api/tickets/{id}": ["tickets:delete"],
    "POST /api/tickets/labels/colors": ["tickets:read"],
    "POST /api/tickets/{id}/labels": ["tickets:write"],
    "POST /api/chat/dms": ["chat:write", "none"],
  };
  const check = (code: string) => problemsIn("hook.tsx", parse("hook.tsx", code), routes, new Map()).problems;

  it.each([
    ["a declared request the table agrees with", 'const f = requires("tickets:write", (id: string) => api.patch(`/api/tickets/${id}/status`, {}));', 0],
    ["a literal route beats a parameter that also fits", 'const f = requires("tickets:read", () => api.post("/api/tickets/labels/colors", {}));', 0],
    ["any permission the route lists", 'const f = requires("none", () => api.post("/api/chat/dms", {}));', 0],
    ["a request with no declaration", "const f = (id: string) => api.delete(`/api/tickets/${id}`);", 1],
    ["a permission the route doesn't take", 'const f = requires("tickets:write", (id: string) => api.delete(`/api/tickets/${id}`));', 1],
    ["a route the table doesn't have", 'const f = requires("tickets:write", () => api.post("/api/tickets/archive"));', 1],
    ["a declaration around no request", 'const f = requires("tickets:write", () => api.get("/api/tickets"));', 1],
    ["a URL held in a constant", 'const f = requires("tickets:write", (id: string) => { const path = `/api/tickets/${id}/status`; return api.patch(path, {}); });', 0],
    ["each branch of a URL choice", 'const f = requires("tickets:write", (b: boolean) => api.post(b ? "/api/tickets/labels/colors" : "/api/tickets/x/labels"));', 1],
    ["a URL the call doesn't show", 'const f = requires("tickets:write", (path: string) => api.patch(path, {}));', 1],
  ])("%s", (_, code, problems) => {
    expect(check(code)).toHaveLength(problems);
  });
});
