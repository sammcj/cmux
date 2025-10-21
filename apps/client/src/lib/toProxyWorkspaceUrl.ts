const MORPH_HOST_REGEX = /^port-(\d+)-morphvm-([^.]+)\.http\.cloud\.morph\.so$/;

interface MorphUrlComponents {
  url: URL;
  morphId: string;
  port: number;
}

function parseMorphUrl(input: string): MorphUrlComponents | null {
  if (!input.includes("morph.so")) {
    return null;
  }

  try {
    const url = new URL(input);
    const match = url.hostname.match(MORPH_HOST_REGEX);

    if (!match) {
      return null;
    }

    const [, portString, morphId] = match;
    const port = Number.parseInt(portString, 10);

    if (Number.isNaN(port)) {
      return null;
    }

    return {
      url,
      morphId,
      port,
    };
  } catch {
    return null;
  }
}

function createMorphPortUrl(
  components: MorphUrlComponents,
  port: number
): URL {
  const url = new URL(components.url.toString());
  url.hostname = `port-${port}-morphvm-${components.morphId}.http.cloud.morph.so`;
  return url;
}

export function toProxyWorkspaceUrl(workspaceUrl: string): string {
  const components = parseMorphUrl(workspaceUrl);

  if (!components) {
    return workspaceUrl;
  }

  const scope = "base"; // Default scope
  const proxiedUrl = new URL(components.url.toString());
  proxiedUrl.hostname = `cmux-${components.morphId}-${scope}-${components.port}.cmux.app`;
  return proxiedUrl.toString();
}

export function toMorphVncUrl(sourceUrl: string): string | null {
  const components = parseMorphUrl(sourceUrl);

  if (!components) {
    return null;
  }

  const vncUrl = createMorphPortUrl(components, 39380);
  vncUrl.pathname = "/vnc.html";

  const searchParams = new URLSearchParams();
  searchParams.set("autoconnect", "1");
  searchParams.set("resize", "scale");
  vncUrl.search = `?${searchParams.toString()}`;
  vncUrl.hash = "";

  return vncUrl.toString();
}

export function toWorkspaceServiceUrl(
  workspaceUrl: string,
  options: { port: number; path?: string; protocol?: "http" | "https" },
): string | null {
  const { port, protocol, path = "/" } = options;
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const components = parseMorphUrl(workspaceUrl);

  if (components) {
    const serviceUrl = new URL(components.url.toString());
    serviceUrl.hostname = `cmux-${components.morphId}-base-${port}.cmux.app`;
    serviceUrl.port = "";
    serviceUrl.pathname = normalizedPath;
    serviceUrl.search = "";
    serviceUrl.hash = "";
    if (protocol) {
      serviceUrl.protocol = protocol;
    }
    return serviceUrl.toString();
  }

  try {
    const url = new URL(workspaceUrl);
    url.port = String(port);
    url.pathname = normalizedPath;
    url.search = "";
    url.hash = "";
    if (protocol) {
      url.protocol = protocol;
    }
    return url.toString();
  } catch {
    return null;
  }
}

export function resolveWorkspaceServiceBases(
  workspaceUrl: string,
  port: number,
): string[] {
  const results = new Set<string>();

  const add = (value: string | null | undefined) => {
    if (!value) return;
    const trimmed = value.replace(/\/$/, "");
    if (trimmed.length > 0) {
      results.add(trimmed);
    }
  };

  const components = parseMorphUrl(workspaceUrl);
  if (components) {
    const proxied = new URL(components.url.toString());
    proxied.hostname = `cmux-${components.morphId}-base-${port}.cmux.app`;
    proxied.port = "";
    proxied.pathname = "/";
    proxied.search = "";
    proxied.hash = "";
    add(proxied.toString());

    const direct = new URL(components.url.toString());
    direct.hostname = `port-${port}-morphvm-${components.morphId}.http.cloud.morph.so`;
    direct.port = "";
    direct.pathname = "/";
    direct.search = "";
    direct.hash = "";
    add(direct.toString());
  }

  try {
    const url = new URL(workspaceUrl);
    url.port = String(port);
    url.pathname = "/";
    url.search = "";
    url.hash = "";
    add(url.toString());
  } catch {
    // ignore invalid URL
  }

  return Array.from(results);
}

export function toMorphXtermBaseUrl(sourceUrl: string): string | null {
  const components = parseMorphUrl(sourceUrl);

  if (!components) {
    return null;
  }

  const scope = "base";
  const proxiedUrl = new URL(components.url.toString());
  proxiedUrl.hostname = `cmux-${components.morphId}-${scope}-39383.cmux.app`;
  proxiedUrl.port = "";
  proxiedUrl.pathname = "/";
  proxiedUrl.search = "";
  proxiedUrl.hash = "";

  return proxiedUrl.toString();
}
