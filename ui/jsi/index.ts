
// Go bridge / runtime globals (injected by runtime/vm.go)
declare const __JSI__: {
    getPosts: () => Post[];
    getPost: (slug: string) => Post;
    getProjects: () => Project[];
    getProject: (id: string) => Project;
    getProfile: (id?: string) => Profile;
    getRevisions: (slug: string) => Revision[];
    getRevision: (slug: string, rev: string) => Revision;
    triggerError: (message?: string) => Promise<never>;
};

// Per-request context injected by runtime/vm.go before each render.
declare global {
    const __request__: {
        url: string;
        path: string;
        query: string;
        method: string;
        headers: Record<string, string>;
    };
}

export declare function fetchJSON(url: string): unknown;

// Server entry (injected by build)
export declare const __CLIENT_MANIFEST__: Record<string, {
    id: string;
    name: string;
    chunks: string[];
    async: boolean;
}>;

export interface Post {
    id: number;
    slug: string;
    title: string;
    excerpt: string;
    author: string;
    date: string;
    readTime: number;
    tags: string[];
}

export interface Project {
    id: string;
    title: string;
    description: string;
    tech: string[];
    stars: number;
    status: 'active' | 'stable' | 'beta';
}

export interface Revision {
    rev: string;
    date: string;
    summary: string;
}

export interface Profile {
    id: string;
    name: string;
    email: string;
    role: string;
    bio: string;
    github: string;
    website: string;
}

export default { ...__JSI__ }
