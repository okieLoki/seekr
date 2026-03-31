export const BULK_PASTE_CHAR_LIMIT = 250000;
export const BULK_FILE_CHAR_LIMIT = 2000000;

export function summarizeJsonError(err) {
    if (!err) return 'Invalid JSON.';
    if (err instanceof SyntaxError && err.message) return err.message;
    return String(err.message || err);
}

export function validateBulkJsonText(text, limit = BULK_PASTE_CHAR_LIMIT) {
    const trimmed = text.trim();
    if (!trimmed) {
        return { ok: false, error: 'Bulk import cannot be empty.' };
    }
    if (trimmed.length > limit) {
        return { ok: false, error: `Bulk import exceeds the ${limit.toLocaleString()} character paste limit.` };
    }

    let payload;
    try {
        payload = JSON.parse(trimmed);
    } catch (err) {
        return { ok: false, error: `Invalid JSON: ${summarizeJsonError(err)}` };
    }

    if (!Array.isArray(payload)) {
        return { ok: false, error: 'Bulk import must be a JSON array.' };
    }
    if (!payload.length) {
        return { ok: false, error: 'Bulk import array cannot be empty.' };
    }

    return { ok: true, payload };
}

export async function readBulkJsonFile(file) {
    if (!file) {
        return { ok: false, error: 'Choose a JSON file to load.' };
    }
    const filename = String(file.name || '').toLowerCase();
    if (filename && !filename.endsWith('.json') && file.type !== 'application/json') {
        return { ok: false, error: 'Only .json files are supported.' };
    }

    const text = await file.text();
    if (text.length > BULK_FILE_CHAR_LIMIT) {
        return { ok: false, error: `JSON file is too large. Limit is ${BULK_FILE_CHAR_LIMIT.toLocaleString()} characters.` };
    }

    const result = validateBulkJsonText(text, BULK_FILE_CHAR_LIMIT);
    if (!result.ok) {
        return result;
    }

    return {
        ok: true,
        text: JSON.stringify(result.payload, null, 2),
        payload: result.payload,
    };
}
