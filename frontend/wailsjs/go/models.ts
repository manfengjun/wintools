export namespace desktoplock {
	
	export class BackupResult {
	    ok: number;
	    skipped: number;
	    backup_dir?: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.skipped = source["skipped"];
	        this.backup_dir = source["backup_dir"];
	    }
	}
	export class RestoreResult {
	    restored: number;
	    skipped: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.restored = source["restored"];
	        this.skipped = source["skipped"];
	        this.error = source["error"];
	    }
	}
	export class StatusResult {
	    locked: boolean;
	    backup_count: number;
	    desktop_count: number;
	    missing?: string[];
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locked = source["locked"];
	        this.backup_count = source["backup_count"];
	        this.desktop_count = source["desktop_count"];
	        this.missing = source["missing"];
	    }
	}

}

export namespace pyenv {
	
	export class InstallStatus {
	    installed: boolean;
	    version: string;
	    python_exe: string;
	    pip_installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InstallStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.python_exe = source["python_exe"];
	        this.pip_installed = source["pip_installed"];
	    }
	}
	export class PackageInfo {
	    id: string;
	    name: string;
	    description: string;
	    default_on: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PackageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.default_on = source["default_on"];
	    }
	}
	export class ProgressInfo {
	    step: string;
	    message: string;
	    percent: number;
	    done: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProgressInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.message = source["message"];
	        this.percent = source["percent"];
	        this.done = source["done"];
	        this.error = source["error"];
	    }
	}

}

