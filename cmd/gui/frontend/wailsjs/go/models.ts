export namespace api {
	
	export class PlaylistThumb {
	    photo_300: string;
	    photo_600: string;
	    photo_68: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaylistThumb(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.photo_300 = source["photo_300"];
	        this.photo_600 = source["photo_600"];
	        this.photo_68 = source["photo_68"];
	    }
	}
	export class Playlist {
	    id: number;
	    owner_id: number;
	    title: string;
	    count: number;
	    thumb?: PlaylistThumb;
	    photo?: PlaylistThumb;
	
	    static createFrom(source: any = {}) {
	        return new Playlist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.owner_id = source["owner_id"];
	        this.title = source["title"];
	        this.count = source["count"];
	        this.thumb = this.convertValues(source["thumb"], PlaylistThumb);
	        this.photo = this.convertValues(source["photo"], PlaylistThumb);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

