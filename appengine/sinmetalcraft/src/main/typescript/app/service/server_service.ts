namespace SinmetalCraft {
    "use strict";

    import IStorageBody = SinmetalCraft.Model.IStorageBody;
    import ICursorList = SinmetalCraft.Model.ICursorList;
    import IMessage = SinmetalCraft.Model.IMessage;
    import IListOptions = SinmetalCraft.Model.IListOptions;
    import IServerInsertRequest = SinmetalCraft.Model.IServerInsertRequest;
    import IServerUpdateRequest = SinmetalCraft.Model.IServerUpdateRequest;

    export class ServerService {
        PATH = "/api/1/server";

        /**
         * constructor
         * @param $http
         */
        constructor(public $http:ng.IHttpService, public store:angular.a0.storage.IStoreService) {
        }

        insert(param:IServerInsertRequest):ng.IHttpPromise<IMessage> {
            return this.$http.post(this.PATH, param);
        }

        update(param:IServerUpdateRequest):ng.IHttpPromise<IMessage> {
            return this.$http.put(this.PATH, param);
        }
    }
}
