import type { Post, Revision } from "@/jsi";

const jsi = __JSI__ as {
    getPosts: () => Promise<Post[]>;
    getPost: (slug: string) => Promise<Post>;
    getRevisions: (slug: string) => Promise<Revision[]>;
    getRevision: (slug: string, rev: string) => Promise<Revision>;
    triggerError: (message?: string) => Promise<never>;
};

export default jsi;
