namespace SinmetalCraft {
    "use strict";

    import IStorageBody = SinmetalCraft.Model.IStorageBody;
    import ICursorList = SinmetalCraft.Model.ICursorList;
    import IExample = SinmetalCraft.Model.IExample;
    import IListOptions = SinmetalCraft.Model.IListOptions;
    import IExampleInsertRequest = SinmetalCraft.Model.IExampleInsertRequest;
    import IExampleUpdateRequest = SinmetalCraft.Model.IExampleUpdateRequest;
    import IExampleDeleteRequest = SinmetalCraft.Model.IExampleDeleteRequest;

    export interface IExampleStore {
        example:IExample;
        examples:ICursorList<IExample>;
    }

    export class ExampleService {
        PATH = "/api/1/minecraft";

        DRAFT_INSERT_STORAGE_KEY = "example-draft-insert";
        DRAFT_UPDATE_STORAGE_KEY = "example-draft-update";

        exampleStore:IExampleStore;

        /**
         * constructor
         * @param $http
         */
        constructor(public $http:ng.IHttpService, public store:angular.a0.storage.IStoreService) {
            this.exampleStore = {
                "example":null,
                "examples":{
                    "cursor":"",
                    "hasNext":false,
                    "items":[]
                }
            };
        }

        /**
         * Example一覧を取得する
         * @returns {ng.IHttpPromise<IExample[]>}
         */
        list(options?:IListOptions):ng.IHttpPromise<IExample[]> {
            var opts:any = {};
            var config:any = {};
            if (options) {
                opts.limit = options.limit;
                if (options.cursor) {
                    config.headers = {
                        "X-SinmetalCraft-Cursor" : options.cursor
                    };
                }
            }
            return this.$http.get(this.PATH + "?" + $.param(opts), config);
        }

        get(key:string):ng.IHttpPromise<IExample> {
            return this.$http.get(this.PATH + "?key=" + key);
        }

        insert(param:IExampleInsertRequest):ng.IHttpPromise<IExample> {
            return this.$http.post(this.PATH, param);
        }

        update(param:IExampleUpdateRequest):ng.IHttpPromise<IExample> {
            return this.$http.put(this.PATH, param);
        }

        delete(param:IExampleDeleteRequest):ng.IHttpPromise<IExample> {
            return this.$http.delete(this.PATH, {
                params: param
            });
        }

        setDraftForInsert(param:IExampleInsertRequest) {
            var body: IStorageBody = {
                expire: new Date().getUTCDate() + 3600,
                body: param
            };
            this.store.set(this.DRAFT_INSERT_STORAGE_KEY, body);
        }

        getDraftForInsert():IExampleInsertRequest {
            var body: IStorageBody = this.store.get(this.DRAFT_INSERT_STORAGE_KEY);
            if (!body) {
                return {
                    name: ""
                };
            }
            if (body.expire < new Date().getUTCDate()) {
                return {
                    name: ""
                };
            }
            return body.body;
        }

        removeDraftForInsert() {
            this.store.remove(this.DRAFT_INSERT_STORAGE_KEY);
        }

        setDraftForUpdate(param:IExampleUpdateRequest) {
            var body: IStorageBody = {
                expire: new Date().getUTCDate() + 3600,
                body: param
            };
            this.store.set(this.DRAFT_UPDATE_STORAGE_KEY, body);
        }

        getDraftForUpdate():IExampleUpdateRequest {
            var body: IStorageBody = this.store.get(this.DRAFT_UPDATE_STORAGE_KEY);
            if (!body) {
                return {
                    key: "",
                    name: ""
                };
            }
            if (body.expire < new Date().getUTCDate()) {
                return {
                    key: "",
                    name: ""
                };
            }
            return body.body;
        }

        removeDraftForUpdate() {
            this.store.remove(this.DRAFT_UPDATE_STORAGE_KEY);
        }
    }
}
