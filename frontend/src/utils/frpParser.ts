export interface ParsedFRPConfig {
  serverAddr?: string;
  serverPort?: number;
  user?: string;
  token?: string;
  proxyName?: string;
  remotePort?: number;
  protocol?: string;
  tlsEnable?: boolean;
  disableCustomTLSFirstByte?: boolean;
}

/**
 * Parses FRP client configuration text (supports both TOML and INI formats),
 * extracting parameters commonly used by OpenFrp and generic FRP providers.
 */
export function parseFRPConfigText(rawText: string): ParsedFRPConfig {
  const result: ParsedFRPConfig = {};
  if (!rawText || !rawText.trim()) {
    return result;
  }

  const lines = rawText.split(/\r?\n/);
  let currentSection = '';

  for (let line of lines) {
    line = line.trim();
    if (!line || line.startsWith('#') || line.startsWith(';')) {
      continue;
    }

    // Match section headers: [section] or [[section]]
    const sectionMatch = line.match(/^\[+([a-zA-Z0-9_.\-]+)\]+/);
    if (sectionMatch) {
      currentSection = sectionMatch[1].toLowerCase();
      // In INI format, a non-standard section is often the proxy name (e.g. [test] or [web])
      if (
        !['common', 'auth', 'transport', 'transport.tls', 'proxies', 'plugin'].includes(
          currentSection
        )
      ) {
        result.proxyName = sectionMatch[1];
      }
      continue;
    }

    // Match key-value: key = value or key: value
    const eqIdx = line.indexOf('=');
    const colIdx = line.indexOf(':');
    let sepIdx = -1;
    if (eqIdx !== -1 && colIdx !== -1) {
      sepIdx = Math.min(eqIdx, colIdx);
    } else if (eqIdx !== -1) {
      sepIdx = eqIdx;
    } else if (colIdx !== -1) {
      sepIdx = colIdx;
    }

    if (sepIdx === -1) {
      continue;
    }

    const rawKey = line.slice(0, sepIdx).trim();
    let rawVal = line.slice(sepIdx + 1).trim();

    // Strip inline comments: "value # comment" or "value ; comment"
    const commentMatch = rawVal.match(/^(.*?)(?:\s+[#;].*)$/);
    if (commentMatch) {
      rawVal = commentMatch[1].trim();
    }

    // Strip wrapping single or double quotes
    if (
      (rawVal.startsWith("'") && rawVal.endsWith("'")) ||
      (rawVal.startsWith('"') && rawVal.endsWith('"'))
    ) {
      rawVal = rawVal.slice(1, -1).trim();
    }

    const keyLower = rawKey.toLowerCase().replace(/[-_]/g, '');
    const valLower = rawVal.toLowerCase();

    // Key mappings
    if (keyLower === 'serveraddr') {
      result.serverAddr = rawVal;
    } else if (keyLower === 'serverport') {
      const port = parseInt(rawVal, 10);
      if (!isNaN(port)) result.serverPort = port;
    } else if (keyLower === 'user') {
      result.user = rawVal;
    } else if (keyLower === 'token') {
      result.token = rawVal;
    } else if (keyLower === 'name' || (currentSection === 'proxies' && keyLower === 'name')) {
      result.proxyName = rawVal;
    } else if (keyLower === 'remoteport') {
      const port = parseInt(rawVal, 10);
      if (!isNaN(port)) result.remotePort = port;
    } else if (keyLower === 'protocol' || (currentSection === 'transport' && keyLower === 'protocol')) {
      result.protocol = rawVal;
    } else if (
      (currentSection === 'transport.tls' && keyLower === 'enable') ||
      keyLower === 'tlsenable'
    ) {
      result.tlsEnable = valLower === 'true' || valLower === '1';
    } else if (
      keyLower === 'disablecustomtlsfirstbyte' ||
      (currentSection === 'transport.tls' && keyLower === 'disablecustomtlsfirstbyte')
    ) {
      result.disableCustomTLSFirstByte = valLower === 'true' || valLower === '1';
    }
  }

  // Automatic heuristic: OpenFrp nodes (e.g. ofalias.net) typically require TLS with custom byte preserved
  if (result.serverAddr && result.serverAddr.includes('ofalias.net')) {
    if (result.tlsEnable === undefined) {
      result.tlsEnable = true;
    }
    if (result.disableCustomTLSFirstByte === undefined) {
      result.disableCustomTLSFirstByte = false;
    }
  }

  return result;
}
