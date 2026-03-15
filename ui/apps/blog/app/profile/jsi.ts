import type { Profile } from "@/jsi";

const jsi = __JSI__ as {
    getProfile: (id?: string) => Promise<Profile>;
    triggerError: (message?: string) => Promise<never>;
};

export default jsi;
