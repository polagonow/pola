import type { Post, Project } from "@/jsi";

const jsi = __JSI__ as {
    getPosts: () => Promise<Post[]>;
    getProjects: () => Promise<Project[]>;
    triggerError: (message?: string) => Promise<never>;
};

export default jsi;
