// File helpers for the upload flow. Images become data URLs so they persist
// in localStorage across reloads (the mock has no real object storage — that
// arrives in Phase 2 with S3).

export function fileToDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(new Error('Could not read file'));
    reader.readAsDataURL(file);
  });
}

export function bytesToMB(bytes: number): number {
  return Math.max(1, Math.round((bytes / (1024 * 1024)) * 10) / 10);
}
