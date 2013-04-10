var Directory = Backbone.Model.extend({
    idAttribute: "Name"
});

var DirectoryView = Backbone.View.extend({
    events: {
        "click li.directory a": "clickDirectory"
    },
    initialize: function(options) {
        this.parent = options.parent;
        this.childViews = {};
    },
    render: function() {
        this.$el.html('');
        if(this.parent) {
            this.$el.append("<li class='directory'><a href='#' data-target='..'><i class='icon-folder-close'></i>..</a></li>");
        }
        _.each(this.model.get('Children'), function(dir) {
            this.$el.append("<li class='directory'><a href='#' data-target='" + dir.Name + "'><i class='icon-folder-close'></i>" + dir.Name + "</a></li>");
        }, this);
        _.each(this.model.get('Files'), function(file) {
            this.$el.append("<li class='file' data-path='" + file.Path + "'><i class='icon-file'></i><label class='checkbox'><input type='checkbox'>" + file.Name + "</label></li>");
        }, this);
        return this;
    },
    clickDirectory: function(evt) {
        var targetName = $(evt.currentTarget).attr('data-target');

        if(targetName == ".." && this.parent) {
            this.undelegateEvents();
            this.parent.render();
            this.parent.delegateEvents();
            return false;
        }

        var subdir = _.find(this.model.get('Children'), function(dir) {
            return dir.Name == targetName
        }, this);
        if(!subdir) return;

        if(!this.childViews[targetName]) {
            this.childViews[targetName] = new DirectoryView({
                model: new Directory(subdir),
                el: this.el,
                parent: this
            });
        }
        this.undelegateEvents();
        this.childViews[targetName].render();
        this.childViews[targetName].delegateEvents();
        return false;
    }
});

var QueueItem = Backbone.Model.extend({
    idAttribute: "Name",
    defaults: {
        progress: 0
    }
});

var Queue = Backbone.Collection.extend({
    model: QueueItem,
    url: "/api/queue"
});

var QueueItemView = Backbone.View.extend({
    initialize: function() {
        this.template = Handlebars.compile($('#queue-item-template').html());
        this.model.on("change:progress", this.updateProgress, this);
    },
    render: function() {
        var html = this.template(this.model.toJSON());
        this.$el.html(html);
        return this;
    },
    updateProgress: function() {
        this.$(".bar").css("width", this.model.get('progress') + "%")
    }
});

var QueueView = Backbone.View.extend({
    initialize: function() {
        this.collection.on("change add remove", this.render, this);
    },
    render: function() {
        this.$el.html('');
        if(this.collection.length == 0) {
            this.$el.html("<li>Nothing queued</li>");
            return this;
        }

        this.collection.each(function(item) {
            var view = new QueueItemView({
                model: item,
                tagName: "li"
            });
            this.$el.append(view.render().el);
        }, this);
        return this;
    }
});

var HandbrakeRouter = Backbone.Router.extend({
    routes: {
        "":"default",
    },
    default: function() {
        $.getJSON("/api/files/source", function(data) {
            var root = new Directory(data);
            var view = new DirectoryView({
                model: root,
                el: $("ul.files")
            });
            view.render();
        });

        var router = this;

        $("button#queue").on("click", function() { router.queueFiles()});

        this.queue = new Queue;
        this.queue.fetch({
            success: function() {
                var view = new QueueView({
                    collection: router.queue,
                    el: $("ul.queue")
                });
                view.render();
            }
        });

        window.setInterval(function() {
            router.queue.fetch();
        }, 30000);

        if (window["WebSocket"]) {
            conn = new WebSocket("ws://" + window.location.host + "/api/queue/status");
            conn.onclose = function(evt) {
                console.log("Connection closed.")
            }
            conn.onmessage = function(evt) {
                var status = JSON.parse(evt.data);
                var item = router.queue.get(status.Name);
                if(!item) return;
                item.set({progress:status.Progress.toFixed(2)});
            }
        } else {
            console.debug("websockets not supported");
        }
    },
    queueFiles: function() {
        var router = this;
        $("li.file").each(function(idx, el) {
            var checked = $("input", el).prop('checked');
            if(!checked) return;
            var path = $(el).attr('data-path');
            $.ajax({
                type: "POST", 
                url: "/api/queue", 
                data: '{"Path":"' + path + '"}', 
                contentType: "application/json",
                success: function() {
                    console.debug("Queued " + path);
                    router.queue.fetch();
                }
            });
        });
    }
});