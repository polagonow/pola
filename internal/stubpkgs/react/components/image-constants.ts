// Pola's image backend (the "imaging" adapter) produces a single output format
// Candidate widths used to build responsive srcSet entries when an explicit
// width is not supplied.
export const DEFAULT_DEVICE_SIZES = [640, 750, 828, 1080, 1200, 1920, 2048, 3840];

// Smaller widths intended for fixed-size images and icons.
export const DEFAULT_IMAGE_SIZES = [16, 32, 48, 64, 96, 128, 256, 384];

// Quality levels the optimizer is expected to honor (informational).
export const DEFAULT_QUALITY_LEVELS = [25, 50, 75, 100];
