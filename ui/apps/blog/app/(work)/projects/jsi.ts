import type { Project } from "@/jsi";

const jsi = __JSI__ as {
    getProjects: () => Promise<Project[]>;
    getProject: (id: string) => Promise<Project>;
    triggerError: (message?: string) => Promise<never>;
};

export default jsi;
